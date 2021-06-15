import { BigNumber, Contract, ContractFactory, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import chai, { expect } from 'chai'
import { GWEI, fundUser, encodeSolidityRevertMessage } from './shared/utils'
import { OptimismEnv } from './shared/env'
import { solidity } from 'ethereum-waffle'
import { sleep } from '../../packages/core-utils/dist'
import {
  getContractFactory,
  getContractInterface,
} from '../../packages/contracts/dist'
import { Interface } from 'ethers/lib/utils'

chai.use(solidity)

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

    const checkBalances = async (
      expectedBalances: BigNumber[]
    ): Promise<void> => {
      const realBalances = await getBalances()
      expect(realBalances[0]).to.deep.eq(expectedBalances[0])
      expect(realBalances[1]).to.deep.eq(expectedBalances[1])
    }

    const value = 10
    await fundUser(env.watcher, env.l1Bridge, value, wallet.address)

    const initialBalances = await getBalances()

    const there = await wallet.sendTransaction({
      to: other.address,
      value,
      gasPrice: 0,
    })
    await there.wait()

    await checkBalances([
      initialBalances[0].sub(value),
      initialBalances[1].add(value),
    ])

    const backAgain = await other.sendTransaction({
      to: wallet.address,
      value,
      gasPrice: 0,
    })
    await backAgain.wait()

    await checkBalances(initialBalances)
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
      const ovmEthBalanceOf0 = await env.ovmEth.balanceOf(ValueCalls0.address)
      const ovmEthBalanceOf1 = await env.ovmEth.balanceOf(ValueCalls1.address)
      expect(ovmEthBalanceOf0).to.deep.eq(
        BigNumber.from(expectedBalances[0]),
        'geth RPC does not match OVM_ETH.balanceOf'
      )
      expect(ovmEthBalanceOf1).to.deep.eq(
        BigNumber.from(expectedBalances[1]),
        'geth RPC does not match OVM_ETH.balanceOf'
      )
      // query address(this).balance solidity via eth_call as final check
      const ovmAddressThisBalance0 = await ValueCalls0.callStatic.getAddressThisBalance()
      const ovmAddressThisBalance01 = await ValueCalls1.callStatic.getAddressThisBalance()
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
      await fundUser(
        env.watcher,
        env.l1Bridge,
        initialBalance0,
        ValueCalls0.address
      )
      // These tests ass assume ValueCalls0 starts with a balance, but ValueCalls1 does not.
      await checkBalances([initialBalance0, 0])
    })

    it('should allow ETH to be sent', async () => {
      const sendAmount = 15
      const tx = await ValueCalls0.simpleSend(ValueCalls1.address, sendAmount, {
        gasPrice: 0,
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
      await fundUser(
        env.watcher,
        env.l1Bridge,
        initialBalance,
        ValueCalls1.address
      )

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

      const [
        outerSuccess,
        outerReturndata,
      ] = await ValueCalls0.callStatic.sendWithData(
        ValueCalls1.address,
        sendAmount,
        ValueCalls1.interface.encodeFunctionData('delegateCallToCallValue', [
          ValueContext.address,
        ])
      )
      const [
        innerSuccess,
        innerReturndata,
      ] = ValueCalls1.interface.decodeFunctionResult(
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

      const [
        delegatedSuccess,
        delegatedReturndata,
      ] = await ValueCalls0.callStatic.delegateCallToAddressThisBalance(
        ValueContext.address
      )

      expect(delegatedSuccess).to.be.true
      expect(delegatedReturndata).to.deep.eq(BigNumber.from(initialBalance0))
    })

    it('should have correct address(this).balance through ovmDELEGATECALLs to same account', async () => {
      const [
        delegatedSuccess,
        delegatedReturndata,
      ] = await ValueCalls0.callStatic.delegateCallToAddressThisBalance(
        ValueCalls0.address
      )

      expect(delegatedSuccess).to.be.true
      expect(delegatedReturndata).to.deep.eq(BigNumber.from(initialBalance0))
    })

    it('should allow delegate calls which preserve msg.value even with no balance going into the inner call', async () => {
      const Factory__SendETHAwayAndDelegateCall: ContractFactory = await ethers.getContractFactory(
        'SendETHAwayAndDelegateCall',
        wallet
      )
      const SendETHAwayAndDelegateCall: Contract = await Factory__SendETHAwayAndDelegateCall.deploy()
      await SendETHAwayAndDelegateCall.deployTransaction.wait()

      const value = 17
      const [
        delegatedSuccess,
        delegatedReturndata,
      ] = await SendETHAwayAndDelegateCall.callStatic.emptySelfAndDelegateCall(
        ValueCalls0.address,
        ValueCalls0.interface.encodeFunctionData('getCallValue'),
        {
          value,
        }
      )

      expect(delegatedSuccess).to.be.true
      expect(delegatedReturndata).to.deep.eq(BigNumber.from(value))
    })

    describe('Intrinsic gas for ovmCALL types', async () => {
      let CALL_WITH_VALUE_INTRINSIC_GAS
      let ValueGasMeasurer: Contract
      before(async () => {
        // Grab public variable from the EM
        const OVM_ExecutionManager = new Contract(
          await env.addressManager.getAddress('OVM_ExecutionManager'),
          getContractInterface('OVM_ExecutionManager', false),
          env.l1Wallet.provider
        )
        const CALL_WITH_VALUE_INTRINSIC_GAS_BIGNUM = await OVM_ExecutionManager.CALL_WITH_VALUE_INTRINSIC_GAS()
        CALL_WITH_VALUE_INTRINSIC_GAS = CALL_WITH_VALUE_INTRINSIC_GAS_BIGNUM.toNumber()

        const Factory__ValueGasMeasurer = await ethers.getContractFactory(
          'ValueGasMeasurer',
          wallet
        )
        ValueGasMeasurer = await Factory__ValueGasMeasurer.deploy()
        await ValueGasMeasurer.deployTransaction.wait()
      })

      it('a call with value to an empty account consumes <= the intrinsic gas including a buffer', async () => {
        const value = 1
        const gasLimit = 1_000_000
        const minimalSendGas = await ValueGasMeasurer.callStatic.measureGasOfTransferingEthViaCall(
          ethers.constants.AddressZero,
          value,
          gasLimit,
          {
            gasLimit: 2_000_000,
          }
        )

        const buffer = 1.2
        expect(minimalSendGas * buffer).to.be.lte(CALL_WITH_VALUE_INTRINSIC_GAS)
      })

      it('a call with value to an reverting account consumes <= the intrinsic gas including a buffer', async () => {
        // [magic deploy prefix] . [MSTORE] (will throw exception from no stack args)
        const AutoRevertInitcode = '0x600D380380600D6000396000f3' + '52'
        const Factory__AutoRevert = new ContractFactory(
          new Interface([]),
          AutoRevertInitcode,
          wallet
        )
        const AutoRevert: Contract = await Factory__AutoRevert.deploy()
        await AutoRevert.deployTransaction.wait()

        const value = 1
        const gasLimit = 1_000_000
        // A revert, causing the ETH to be sent back, should consume the minimal possible gas for a nonzero ETH send
        const minimalSendGas = await ValueGasMeasurer.callStatic.measureGasOfTransferingEthViaCall(
          AutoRevert.address,
          value,
          gasLimit,
          {
            gasLimit: 2_000_000,
          }
        )

        const buffer = 1.2
        expect(minimalSendGas * buffer).to.be.lte(CALL_WITH_VALUE_INTRINSIC_GAS)
      })

      it('a value call passing less than the intrinsic gas should appear to revert', async () => {
        const Factory__PayableConstant: ContractFactory = await ethers.getContractFactory(
          'PayableConstant',
          wallet
        )
        const PayableConstant: Contract = await Factory__PayableConstant.deploy()
        await PayableConstant.deployTransaction.wait()

        const sendAmount = 15
        const [
          success,
          returndata,
        ] = await ValueCalls0.callStatic.sendWithDataAndGas(
          PayableConstant.address,
          sendAmount,
          PayableConstant.interface.encodeFunctionData('returnValue'),
          CALL_WITH_VALUE_INTRINSIC_GAS - 1,
          {
            gasLimit: 2_000_000,
          }
        )

        expect(success).to.eq(false)
        expect(returndata).to.eq('0x')
      })

      it('a value call which runs out of gas does not out-of-gas the parent', async () => {
        const Factory__TestOOG: ContractFactory = await ethers.getContractFactory(
          'TestOOG',
          wallet
        )
        const TestOOG: Contract = await Factory__TestOOG.deploy()
        await TestOOG.deployTransaction.wait()

        const sendAmount = 15
        // Implicitly test that this call is not rejected
        const [
          success,
          returndata,
        ] = await ValueCalls0.callStatic.sendWithDataAndGas(
          TestOOG.address,
          sendAmount,
          TestOOG.interface.encodeFunctionData('runOutOfGas'),
          CALL_WITH_VALUE_INTRINSIC_GAS * 2,
          {
            gasLimit: 2_000_000,
          }
        )

        expect(success).to.eq(false)
        expect(returndata).to.eq('0x')
      })
    })
  })
})
