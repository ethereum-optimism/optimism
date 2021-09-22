import { expect } from 'chai'

/* Imports: External */
import { ethers } from 'hardhat'
import { Contract, ContractFactory } from 'ethers'

/* Imports: Internal */
import {
  Direction,
  OptimismEnv,
  useDynamicTimeoutForWithdrawals,
  DEFAULT_SENDER_ADDRESS,
} from './shared'

describe('Basic L1<>L2 Communication', async () => {
  let env: OptimismEnv
  before(async () => {
    env = await OptimismEnv.new()
  })

  let Factory__SimpleStorage: ContractFactory
  let Factory__Reverter: ContractFactory
  before(async () => {
    Factory__SimpleStorage = await ethers.getContractFactory('SimpleStorage')
    Factory__Reverter = await ethers.getContractFactory('Reverter')
  })

  let L1SimpleStorage: Contract
  let L2SimpleStorage: Contract
  let L2Reverter: Contract
  beforeEach(async () => {
    // Deploy SimpleStorage on L1.
    L1SimpleStorage = await Factory__SimpleStorage.connect(
      env.l1Wallet
    ).deploy()
    await L1SimpleStorage.deployTransaction.wait()

    // Deploy SimpleStorage on L2.
    L2SimpleStorage = await Factory__SimpleStorage.connect(
      env.l2Wallet
    ).deploy()
    await L2SimpleStorage.deployTransaction.wait()

    // Deploy Reverter on L2.
    L2Reverter = await Factory__Reverter.connect(env.l2Wallet).deploy()
    await L2Reverter.deployTransaction.wait()
  })

  describe('L2 => L1', () => {
    it('should be able to send L2 -> L1 via the L2CrossDomainMessenger', async function () {
      await useDynamicTimeoutForWithdrawals(this, env)

      const value = `0x${'77'.repeat(32)}`

      // Send L2 -> L1 message.
      const transaction = await env.l2Messenger.sendMessage(
        L1SimpleStorage.address,
        L1SimpleStorage.interface.encodeFunctionData('setValue', [value]),
        5000000
      )
      await transaction.wait()
      await env.relayXDomainMessages(transaction)
      await env.waitForXDomainTransaction(transaction, Direction.L2ToL1)

      expect(await L1SimpleStorage.msgSender()).to.equal(
        env.l1Messenger.address
      )
      expect(await L1SimpleStorage.xDomainSender()).to.equal(
        env.l2Wallet.address
      )
      expect(await L1SimpleStorage.value()).to.equal(value)
      expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(1)
    })
  })

  describe('L1 => L2', () => {
    it('should be able to send L1 -> L2 via the L1CrossDomainMessenger', async () => {
      const value = `0x${'42'.repeat(32)}`

      // Send L1 -> L2 message.
      const transaction = await env.l1Messenger.sendMessage(
        L2SimpleStorage.address,
        L2SimpleStorage.interface.encodeFunctionData('setValue', [value]),
        5000000
      )

      await env.waitForXDomainTransaction(transaction, Direction.L1ToL2)

      expect(await L2SimpleStorage.msgSender()).to.equal(
        env.l2Messenger.address
      )
      expect(await L2SimpleStorage.xDomainSender()).to.equal(
        env.l1Wallet.address
      )
      expect(await L2SimpleStorage.value()).to.equal(value)
      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(1)
    })

    it('should be able to send L1 -> L2 directly via enqueue', async () => {
      const value = `0x${'42'.repeat(32)}`

      // Send L1 -> L2 message.
      const transaction = await env.ctc.enqueue(
        L2SimpleStorage.address,
        5000000,
        L2SimpleStorage.interface.encodeFunctionData('setValueNotXDomain', [
          value,
        ])
      )

      await env.waitForXDomainTransaction(transaction, Direction.L1ToL2)

      expect(await L2SimpleStorage.msgSender()).to.equal(DEFAULT_SENDER_ADDRESS)
      expect(await L2SimpleStorage.value()).to.equal(value)
      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(1)
    })

    it('should have a receipt with a status of 1 for a successful message', async () => {
      const value = `0x${'42'.repeat(32)}`

      // Send L1 -> L2 message.
      const transaction = await env.l1Messenger.sendMessage(
        L2SimpleStorage.address,
        L2SimpleStorage.interface.encodeFunctionData('setValue', [value]),
        5000000
      )

      const { remoteReceipt } = await env.waitForXDomainTransaction(
        transaction,
        Direction.L1ToL2
      )

      expect(remoteReceipt.status).to.equal(1)
    })

    // SKIP: until we decide what should be done in this case
    it.skip('should have a receipt with a status of 0 for a failed message', async () => {
      // Send L1 -> L2 message.
      const transaction = await env.l1Messenger.sendMessage(
        L2Reverter.address,
        L2Reverter.interface.encodeFunctionData('doRevert', []),
        5000000
      )

      const { remoteReceipt } = await env.waitForXDomainTransaction(
        transaction,
        Direction.L1ToL2
      )

      expect(remoteReceipt.status).to.equal(0)
    })
  })
})
