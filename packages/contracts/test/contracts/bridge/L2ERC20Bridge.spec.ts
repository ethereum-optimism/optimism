import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'
import { MessageChannel } from 'worker_threads'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Tests */
describe('L2ERC20Bridge', () => {
  const provider = ethers.provider

  let depositer: Signer
  let withdrawer: Signer
  let L2ERC20Bridge: ContractFactory
  let DepositedERC20: ContractFactory
  let MockL2ToL1MessagePasser: ContractFactory
  const mockL1ERC20Address = '0x' + '00'.repeat(20)
  const mockL1ERC20BridgeAddress = '0x' + '11'.repeat(20) 

  before(async () => {
    ;[depositer, withdrawer] = await ethers.getSigners()
    L2ERC20Bridge = await ethers.getContractFactory('L2ERC20Bridge')
    DepositedERC20 = await ethers.getContractFactory('DepositedERC20')
    MockL2ToL1MessagePasser = await ethers.getContractFactory(
      'MockL2ToL1MessagePasser'
    )
  })

  let l2ERC20Bridge: Contract
  let depositedERC20: Contract
  let l2ToL1MessagePasser: Contract
  beforeEach(async () => {
    l2ToL1MessagePasser = await MockL2ToL1MessagePasser.deploy()
    l2ERC20Bridge = await L2ERC20Bridge.deploy(
      mockL1ERC20BridgeAddress,
      l2ToL1MessagePasser.address
    ) //some random addy to represent l1ERC20Bridge
    await l2ERC20Bridge.deployNewDepositedERC20(mockL1ERC20Address, 'Token Name', 10, 'Token Symbol')
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
          await l2ERC20Bridge.deployNewDepositedERC20(mockL1ERC20Address, 'Token Name', 10, 'Token Symbol')
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
    it('forwards withdrawal to L1 and increments withdrawal nonce', async () => {
      depositedERC20 = depositedERC20.connect(withdrawer)
      const initialNonce = await l2ERC20Bridge.withdrawalNonce()
      const withdrawTo = '0x' + '22'.repeat(20)
      await depositedERC20.initializeWithdrawal(withdrawTo, 0)
      const newNonce = await l2ERC20Bridge.withdrawalNonce()
      newNonce.should.equal(initialNonce + 1)
    })
  })
})
