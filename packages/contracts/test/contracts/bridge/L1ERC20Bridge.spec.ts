import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'
import { MessageChannel } from 'worker_threads'
import { deployAndRegister } from 'src'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Tests */
describe.only('L1ERC20Bridge', () => {
  const provider = ethers.provider
  let depositer: Signer
  let withdrawer: Signer
  let L1ERC20Bridge: ContractFactory
  let L2ERC20Bridge: ContractFactory
  let MockL2ToL1MessagePasser: ContractFactory
  let MockL1ToL2MessagePasser: ContractFactory
  let ERC20: ContractFactory
  let DepositedERC20: ContractFactory

  before(async () => {
    ;[depositer, withdrawer] = await ethers.getSigners()
    L1ERC20Bridge = await ethers.getContractFactory('L1ERC20Bridge')
    L2ERC20Bridge = await ethers.getContractFactory('L2ERC20Bridge')
    MockL2ToL1MessagePasser = await ethers.getContractFactory(
      'MockL2ToL1MessagePasser'
    )
    MockL1ToL2MessagePasser = await ethers.getContractFactory(
      'MockL1ToL2MessagePasser'
    )
    ERC20 = await (await ethers.getContractFactory('ERC20')).connect(depositer)
    DepositedERC20 = await ethers.getContractFactory('DepositedERC20')
  })

  let l1ERC20Bridge: Contract
  let l2ERC20Bridge: Contract
  let l2ToL1MessagePasser: Contract
  let l1ToL2MessagePasser: Contract
  let wrappedSNX: Contract
  let l2WrappedSNX: Contract
  beforeEach(async () => {
    l1ToL2MessagePasser = await MockL1ToL2MessagePasser.deploy()
    l2ToL1MessagePasser = await MockL2ToL1MessagePasser.deploy()
    /*This is just the mocked L1 to L2 message passing. Should replace in 
    the future with address resolver pattern.*/
    l1ERC20Bridge = await L1ERC20Bridge.deploy(l1ToL2MessagePasser.address)
    l2ERC20Bridge = await L2ERC20Bridge.deploy(
      l1ERC20Bridge.address,
      l2ToL1MessagePasser.address
    )
    //Deploy an ERC20 contract to test deposits and withdrawals
    wrappedSNX = await ERC20.deploy(100, 'Wrapped SNX', 10, 'wSNX')
    await l2ERC20Bridge.deployNewDepositedERC20(
      wrappedSNX.address,
      'Wrapped SNX',
      10,
      'wSNX'
    )
    l2WrappedSNX = DepositedERC20.attach(
      await l2ERC20Bridge.correspondingDepositedERC20(wrappedSNX.address)
    )
  })

  describe('setCorrespondingL2BridgeAddress()', async () => {
    it('Sets address correctly, and throws if address has already been set', async () => {
      await l1ERC20Bridge.setCorrespondingL2BridgeAddress(l2ERC20Bridge.address)
      const l2BridgeAddress = await l1ERC20Bridge.l2ERC20BridgeAddress()
      // Try to set it again
      await TestUtils.assertRevertsAsync(
        'This address has already been set.',
        async () => {
          await l1ERC20Bridge.setCorrespondingL2BridgeAddress(
            '0x' + '00'.repeat(20)
          )
        }
      )
    })
  })

  describe('initializeDeposit()', async () => {
    it('transfers funds to this contract and mints corresponding coins on L2', async () => {
      const initialBalance = await wrappedSNX.balanceOf(l1ERC20Bridge.address)
      const depositAmount = 5
      // Transfer deposit to this contract
      await wrappedSNX.approve(l1ERC20Bridge.address, depositAmount)
      await l1ERC20Bridge.initializeDeposit(
        wrappedSNX.address,
        depositer.getAddress(),
        depositAmount
      )
      const newBalance = await wrappedSNX.balanceOf(l1ERC20Bridge.address)
      newBalance.should.equal(initialBalance + depositAmount)
      // // Check that funds are created on L2
      const l2Balance = await l2WrappedSNX.balanceOf(depositer.getAddress())
      console.log('layer 2 balance is', l2Balance)
      /*
      * This won't work until message passing is finished
      *
      l2Balance.should.equal(depositAmount)
      l2ERC20Bridge.correspondingDepositedERC20[wrappedSNX.address]
        .balanceOf(depositer.getAddress())
        .should.equal(depositAmount)
      */
    })

    describe('redeemWithdrawal()', async () => {
      it('throws if the withdrawal has already been redeemed', async () => {
        // doesn't work rn
      })
    })
  })
})
