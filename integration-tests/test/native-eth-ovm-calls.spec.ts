/* Imports: External */
import { BigNumber, Contract, ContractFactory, Wallet } from 'ethers'
import { ethers } from 'hardhat'

/* Imports: Internal */
import { expect } from './shared/setup'
import {
  fundUser,
  encodeSolidityRevertMessage,
  gasPriceForL2,
} from './shared/utils'
import { OptimismEnv } from './shared/env'

describe('Native ETH value integration tests', () => {
  let env: OptimismEnv
  let wallet: Wallet
  let other: Wallet

  before(async () => {
    env = await OptimismEnv.new()
    wallet = env.l2Wallet
    other = Wallet.createRandom().connect(wallet.provider)
  })

  it('should allow an L2 EOA to send to a new account and back again', async () => {
    const getBalances = async (): Promise<BigNumber[]> => {
      return [
        await wallet.provider.getBalance(wallet.address),
        await wallet.provider.getBalance(other.address),
      ]
    }

    const expectBalancesWithinRange = (
      bal: BigNumber,
      lte: BigNumber,
      gte: BigNumber
    ) => {
      expect(bal.lte(lte)).to.be.true
      expect(bal.gte(gte)).to.be.true
    }

    const value = ethers.utils.parseEther('0.01')
    await fundUser(env.messenger, value, wallet.address)

    const initialBalances = await getBalances()

    const there = await wallet.sendTransaction({
      to: other.address,
      value,
      gasPrice: await gasPriceForL2(),
    })
    const thereReceipt = await there.wait()
    const thereGas = thereReceipt.gasUsed.mul(there.gasPrice)

    const thereBalances = await getBalances()
    const thereWithGas = initialBalances[0].sub(value).sub(thereGas).sub(100000)
    expectBalancesWithinRange(
      thereBalances[0],
      initialBalances[0].sub(value),
      thereWithGas
    )
    expect(initialBalances[1].add(value).eq(thereBalances[1]))

    const backVal = ethers.utils.parseEther('0.005')
    const backAgain = await other.sendTransaction({
      to: wallet.address,
      value: backVal,
      gasPrice: await gasPriceForL2(),
    })
    const backReceipt = await backAgain.wait()
    const backGas = backReceipt.gasUsed.mul(backAgain.gasPrice)

    const backBalances = await getBalances()
    expectBalancesWithinRange(
      backBalances[0],
      initialBalances[0].sub(thereGas).sub(backVal),
      initialBalances[0].sub(thereGas).sub(backVal).sub(200000)
    )
    expectBalancesWithinRange(
      backBalances[1],
      initialBalances[1].add(backVal).sub(backGas),
      initialBalances[1].add(backVal).sub(backGas).sub(200000)
    )
  })

  describe(`calls between OVM contracts with native ETH value and relevant opcodes`, async () => {
    const initialBalance0 = 42000

    let Factory__ValueCalls: ContractFactory
    let ValueCalls0: Contract
    let ValueCalls1: Contract

    const checkBalances = async (expectedBalances: number[]) => {
      // query geth as one check
      const balance0 = await wallet.provider.getBalance(ValueCalls0.address)
      const balance1 = await wallet.provider.getBalance(ValueCalls1.address)
      expect(balance0).to.deep.eq(BigNumber.from(expectedBalances[0]))
      expect(balance1).to.deep.eq(BigNumber.from(expectedBalances[1]))
      // query ovmBALANCE() opcode via eth_call as another check
      const ovmBALANCE0 = await ValueCalls0.callStatic.getBalance(
        ValueCalls0.address
      )
      const ovmBALANCE1 = await ValueCalls0.callStatic.getBalance(
        ValueCalls1.address
      )
      expect(ovmBALANCE0).to.deep.eq(
        BigNumber.from(expectedBalances[0]),
        'geth RPC does not match ovmBALANCE'
      )
      expect(ovmBALANCE1).to.deep.eq(
        BigNumber.from(expectedBalances[1]),
        'geth RPC does not match ovmBALANCE'
      )
      // query ovmSELFBALANCE() opcode via eth_call as another check
      const ovmSELFBALANCE0 = await ValueCalls0.callStatic.getSelfBalance()
      const ovmSELFBALANCE1 = await ValueCalls1.callStatic.getSelfBalance()
      expect(ovmSELFBALANCE0).to.deep.eq(
        BigNumber.from(expectedBalances[0]),
        'geth RPC does not match ovmSELFBALANCE'
      )
      expect(ovmSELFBALANCE1).to.deep.eq(
        BigNumber.from(expectedBalances[1]),
        'geth RPC does not match ovmSELFBALANCE'
      )
      // query ovmSELFBALANCE() opcode via eth_call as another check
      const ovmEthBalanceOf0 =
        await env.messenger.contracts.l2.OVM_ETH.balanceOf(ValueCalls0.address)
      const ovmEthBalanceOf1 =
        await env.messenger.contracts.l2.OVM_ETH.balanceOf(ValueCalls1.address)
      expect(ovmEthBalanceOf0).to.deep.eq(
        BigNumber.from(expectedBalances[0]),
        'geth RPC does not match OVM_ETH.balanceOf'
      )
      expect(ovmEthBalanceOf1).to.deep.eq(
        BigNumber.from(expectedBalances[1]),
        'geth RPC does not match OVM_ETH.balanceOf'
      )
      // query address(this).balance solidity via eth_call as final check
      const ovmAddressThisBalance0 =
        await ValueCalls0.callStatic.getAddressThisBalance()
      const ovmAddressThisBalance01 =
        await ValueCalls1.callStatic.getAddressThisBalance()
      expect(ovmAddressThisBalance0).to.deep.eq(
        BigNumber.from(expectedBalances[0]),
        'geth RPC does not match address(this).balance'
      )
      expect(ovmAddressThisBalance01).to.deep.eq(
        BigNumber.from(expectedBalances[1]),
        'geth RPC does not match address(this).balance'
      )
    }

    before(async () => {
      Factory__ValueCalls = await ethers.getContractFactory(
        'ValueCalls',
        wallet
      )
    })

    beforeEach(async () => {
      ValueCalls0 = await Factory__ValueCalls.deploy()
      ValueCalls1 = await Factory__ValueCalls.deploy()
      await fundUser(env.messenger, initialBalance0, ValueCalls0.address)
      // These tests ass assume ValueCalls0 starts with a balance, but ValueCalls1 does not.
      await checkBalances([initialBalance0, 0])
    })

    it('should allow ETH to be sent', async () => {
      const sendAmount = 15
      const tx = await ValueCalls0.simpleSend(ValueCalls1.address, sendAmount, {
        gasPrice: await gasPriceForL2(),
      })
      await tx.wait()

      await checkBalances([initialBalance0 - sendAmount, sendAmount])
    })

    it('should revert if a function is nonpayable', async () => {
      const sendAmount = 15
      const [success, returndata] = await ValueCalls0.callStatic.sendWithData(
        ValueCalls1.address,
        sendAmount,
        ValueCalls1.interface.encodeFunctionData('nonPayable')
      )

      expect(success).to.be.false
      expect(returndata).to.eq('0x')
    })

    it('should allow ETH to be sent and have the correct ovmCALLVALUE', async () => {
      const sendAmount = 15
      const [success, returndata] = await ValueCalls0.callStatic.sendWithData(
        ValueCalls1.address,
        sendAmount,
        ValueCalls1.interface.encodeFunctionData('getCallValue')
      )

      expect(success).to.be.true
      expect(BigNumber.from(returndata)).to.deep.eq(BigNumber.from(sendAmount))
    })

    it('should have the correct ovmSELFBALANCE which includes the msg.value', async () => {
      // give an initial balance which the ovmCALLVALUE should be added to when calculating ovmSELFBALANCE
      const initialBalance = 10
      await fundUser(env.messenger, initialBalance, ValueCalls1.address)

      const sendAmount = 15
      const [success, returndata] = await ValueCalls0.callStatic.sendWithData(
        ValueCalls1.address,
        sendAmount,
        ValueCalls1.interface.encodeFunctionData('getSelfBalance')
      )

      expect(success).to.be.true
      expect(BigNumber.from(returndata)).to.deep.eq(
        BigNumber.from(initialBalance + sendAmount)
      )
    })

    it('should have the correct callvalue but not persist the transfer if the target reverts', async () => {
      const sendAmount = 15
      const internalCalldata = ValueCalls1.interface.encodeFunctionData(
        'verifyCallValueAndRevert',
        [sendAmount]
      )
      const [success, returndata] = await ValueCalls0.callStatic.sendWithData(
        ValueCalls1.address,
        sendAmount,
        internalCalldata
      )

      expect(success).to.be.false
      expect(returndata).to.eq(encodeSolidityRevertMessage('expected revert'))

      await checkBalances([initialBalance0, 0])
    })

    it('should look like the subcall reverts with no data if value exceeds balance', async () => {
      const sendAmount = initialBalance0 + 1
      const internalCalldata = ValueCalls1.interface.encodeFunctionData(
        'verifyCallValueAndReturn',
        [sendAmount] // this would be correct and return successfuly, IF it could get here
      )
      const [success, returndata] = await ValueCalls0.callStatic.sendWithData(
        ValueCalls1.address,
        sendAmount,
        internalCalldata
      )

      expect(success).to.be.false
      expect(returndata).to.eq('0x')
    })

    it('should preserve msg.value through ovmDELEGATECALLs', async () => {
      const Factory__ValueContext = await ethers.getContractFactory(
        'ValueContext',
        wallet
      )
      const ValueContext = await Factory__ValueContext.deploy()
      await ValueContext.deployTransaction.wait()

      const sendAmount = 10

      const [outerSuccess, outerReturndata] =
        await ValueCalls0.callStatic.sendWithData(
          ValueCalls1.address,
          sendAmount,
          ValueCalls1.interface.encodeFunctionData('delegateCallToCallValue', [
            ValueContext.address,
          ])
        )
      const [innerSuccess, innerReturndata] =
        ValueCalls1.interface.decodeFunctionResult(
          'delegateCallToCallValue',
          outerReturndata
        )
      const delegatedOvmCALLVALUE = ValueContext.interface.decodeFunctionResult(
        'getCallValue',
        innerReturndata
      )[0]

      expect(outerSuccess).to.be.true
      expect(innerSuccess).to.be.true
      expect(delegatedOvmCALLVALUE).to.deep.eq(BigNumber.from(sendAmount))
    })

    it('should have correct address(this).balance through ovmDELEGATECALLs to another account', async () => {
      const Factory__ValueContext = await ethers.getContractFactory(
        'ValueContext',
        wallet
      )
      const ValueContext = await Factory__ValueContext.deploy()
      await ValueContext.deployTransaction.wait()

      const [delegatedSuccess, delegatedReturndata] =
        await ValueCalls0.callStatic.delegateCallToAddressThisBalance(
          ValueContext.address
        )

      expect(delegatedSuccess).to.be.true
      expect(delegatedReturndata).to.deep.eq(BigNumber.from(initialBalance0))
    })

    it('should have correct address(this).balance through ovmDELEGATECALLs to same account', async () => {
      const [delegatedSuccess, delegatedReturndata] =
        await ValueCalls0.callStatic.delegateCallToAddressThisBalance(
          ValueCalls0.address
        )

      expect(delegatedSuccess).to.be.true
      expect(delegatedReturndata).to.deep.eq(BigNumber.from(initialBalance0))
    })

    it('should allow delegate calls which preserve msg.value even with no balance going into the inner call', async () => {
      const Factory__SendETHAwayAndDelegateCall: ContractFactory =
        await ethers.getContractFactory('SendETHAwayAndDelegateCall', wallet)
      const SendETHAwayAndDelegateCall: Contract =
        await Factory__SendETHAwayAndDelegateCall.deploy()
      await SendETHAwayAndDelegateCall.deployTransaction.wait()

      const value = 17
      const [delegatedSuccess, delegatedReturndata] =
        await SendETHAwayAndDelegateCall.callStatic.emptySelfAndDelegateCall(
          ValueCalls0.address,
          ValueCalls0.interface.encodeFunctionData('getCallValue'),
          {
            value,
          }
        )

      expect(delegatedSuccess).to.be.true
      expect(delegatedReturndata).to.deep.eq(BigNumber.from(value))
    })
  })
})
