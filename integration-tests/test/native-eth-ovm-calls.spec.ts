import { BigNumber, Contract, ContractFactory, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import chai, { expect } from 'chai'
import { GWEI, fundUser, encodeSolidityRevertMessage } from './shared/utils'
import { OptimismEnv } from './shared/env'
import { solidity } from 'ethereum-waffle'

chai.use(solidity)

for (const sourceName of ['ValueCallsWithWrapper', 'ValueCallsWithCompiler']) {
  describe(`OVM calls with native ETH value (${sourceName})`, async () => {
    const initialBalance0 = 42000

    let env: OptimismEnv
    let wallet: Wallet
    let other: Wallet
    let Factory__ValueCalls: ContractFactory
    let ValueCalls0: Contract
    let ValueCalls1: Contract

    const checkBalances = async (expectedBalances: number[]) => {
      // query geth as one check
      const balance0 = await wallet.provider.getBalance(ValueCalls0.address)
      const balance1 = await wallet.provider.getBalance(ValueCalls1.address)
      expect(balance0).to.deep.eq(BigNumber.from(expectedBalances[0]))
      expect(balance1).to.deep.eq(BigNumber.from(expectedBalances[1]))
      // also use ovmBALANCE() opcode via eth_call
      const ovmBALANCE0 = await ValueCalls0.callStatic.getBalance(
        ValueCalls0.address
      )
      const ovmBALANCE1 = await ValueCalls0.callStatic.getBalance(
        ValueCalls1.address
      )
      expect(ovmBALANCE0).to.deep.eq(BigNumber.from(expectedBalances[0]))
      expect(ovmBALANCE1).to.deep.eq(BigNumber.from(expectedBalances[1]))
    }

    before(async () => {
      env = await OptimismEnv.new()
      wallet = env.l2Wallet
      other = Wallet.createRandom().connect(ethers.provider)
      Factory__ValueCalls = await ethers.getContractFactory(sourceName, wallet)
    })

    beforeEach(async () => {
      ValueCalls0 = await Factory__ValueCalls.deploy()
      ValueCalls1 = await Factory__ValueCalls.deploy()
      await fundUser(
        env.watcher,
        env.gateway,
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
  })
}
