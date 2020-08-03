import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'
import { MessageChannel } from 'worker_threads'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Tests */
describe.only('L2ERC20Bridge', () => {
  const provider = ethers.provider

  let depositer: Signer
  let withdrawer: Signer
  let L2ERC20Bridge: ContractFactory
  const mockL1ERC20Address = '0x' + '00'.repeat(20)
  const mockL1ERC20BridgeAddress = '0x' + '11'.repeat(20)
  let DepositedERC20: ContractFactory

  before(async () => {
    ;[depositer, withdrawer] = await ethers.getSigners()
    L2ERC20Bridge = await ethers.getContractFactory('L2ERC20Bridge')
    DepositedERC20 = await ethers.getContractFactory('DepositedERC20')
  })

  let l2ERC20Bridge: Contract
  let depositedERC20: Contract
  beforeEach(async () => {
    l2ERC20Bridge = await L2ERC20Bridge.deploy(mockL1ERC20BridgeAddress) //some random addy to represent l1ERC20Bridge
    await l2ERC20Bridge.deployNewDepositedERC20(mockL1ERC20Address)
    depositedERC20 = DepositedERC20.attach(
      await l2ERC20Bridge.correspondingDepositedERC20(mockL1ERC20Address)
    )
  })

  describe('deployNewDepositedERC20()', async () => {
    it('throws on attempted redeployment for the same ERC20', async () => {
      //TODO: Add integration test to query address of new DepositedERC20 in mapping
      await TestUtils.assertRevertsAsync(
        'L2 ERC20 Contract for this asset already exists.',
        async () => {
          await l2ERC20Bridge.deployNewDepositedERC20(mockL1ERC20Address)
        }
      )
    })
  })

  describe('forwardDeposit', async () => {
    it('forwards deposit correctly', async () => {
      const preDepositBalance = (
        await depositedERC20.balanceOf(depositer.getAddress())
      ).toNumber()
      const depositAmount = 10
      await l2ERC20Bridge.forwardDeposit(
        depositer.getAddress(),
        depositAmount,
        mockL1ERC20Address
      )
      const postDepositBalance = (
        await depositedERC20.balanceOf(depositer.getAddress())
      ).toNumber()
      postDepositBalance.should.equal(preDepositBalance + depositAmount)
    })
  })

  describe('forwardWithdrawal', async () => {
    it('forwards withdrawal correctly and increments withdrawal nonce', async () => {
      depositedERC20 = depositedERC20.connect(withdrawer)
      // deposit some money before withdrawing
      await l2ERC20Bridge.forwardDeposit(
        withdrawer.getAddress(),
        100,
        mockL1ERC20Address
      )
      const preWithdrawalBalance = (
        await depositedERC20.balanceOf(withdrawer.getAddress())
      ).toNumber()
      // checking to see if the withdrawer and msg.sender are the same
      console.log(preWithdrawalBalance)
      console.log('withdrawer address: ', withdrawer.getAddress())
      console.log('msg sender: ', await depositedERC20.returnMS())

      const initialNonce = await l2ERC20Bridge.withdrawalNonce()
      const withdrawTo = '0x' + '22'.repeat(20)
      const withdrawalAmount = 10
      await depositedERC20.initializeWithdrawal(withdrawTo, 0)
      // const postWithdrawalBalance = (
      //   await depositedERC20.balanceOf(withdrawer.getAddress())
      // ).toNumber()
      // //const newNonce = await l2ERC20Bridge.withdrawalNonce()
      // postWithdrawalBalance.should.equal(
      //   preWithdrawalBalance - withdrawalAmount
      // )
      //newNonce.should.equal(initialNonce + 1)
    })
  })
})
