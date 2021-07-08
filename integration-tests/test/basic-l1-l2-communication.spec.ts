import { expect } from 'chai'

/* Imports: External */
import { Contract, ContractFactory } from 'ethers'
import { predeploys, getContractInterface } from '@eth-optimism/contracts'

/* Imports: Internal */
import l1SimpleStorageJson from '../artifacts/contracts/SimpleStorage.sol/SimpleStorage.json'
import l2SimpleStorageJson from '../artifacts-ovm/contracts/SimpleStorage.sol/SimpleStorage.json'
import l2ReverterJson from '../artifacts-ovm/contracts/Reverter.sol/Reverter.json'
import { Direction } from './shared/watcher-utils'
import { OptimismEnv, useDynamicTimeoutForWithdrawals } from './shared/env'

describe('Basic L1<>L2 Communication', async () => {
  let Factory__L1SimpleStorage: ContractFactory
  let Factory__L2SimpleStorage: ContractFactory
  let Factory__L2Reverter: ContractFactory
  let L1SimpleStorage: Contract
  let L2SimpleStorage: Contract
  let L2Reverter: Contract
  let env: OptimismEnv

  before(async () => {
    env = await OptimismEnv.new()
    Factory__L1SimpleStorage = new ContractFactory(
      l1SimpleStorageJson.abi,
      l1SimpleStorageJson.bytecode,
      env.l1Wallet
    )
    Factory__L2SimpleStorage = new ContractFactory(
      l2SimpleStorageJson.abi,
      l2SimpleStorageJson.bytecode,
      env.l2Wallet
    )
    Factory__L2Reverter = new ContractFactory(
      l2ReverterJson.abi,
      l2ReverterJson.bytecode,
      env.l2Wallet
    )
  })

  beforeEach(async () => {
    L1SimpleStorage = await Factory__L1SimpleStorage.deploy()
    await L1SimpleStorage.deployTransaction.wait()
    L2SimpleStorage = await Factory__L2SimpleStorage.deploy()
    await L2SimpleStorage.deployTransaction.wait()
    L2Reverter = await Factory__L2Reverter.deploy()
    await L2Reverter.deployTransaction.wait()
  })

  describe('L2 => L1', () => {
    it('should be able to perform a withdrawal from L2 -> L1', async function () {
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
    it('should deposit from L1 -> L2', async () => {
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

    it('should have a receipt with a status of 0 for a failed message', async () => {
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

    it('should have a receipt with a status of 0 for messages sent to 0x42... addresses', async () => {
      // This call is fine but will give a status of 0.
      const transaction = await env.l1Messenger.sendMessage(
        predeploys.Lib_AddressManager,
        getContractInterface('Lib_AddressManager').encodeFunctionData(
          'getAddress',
          ['whatever']
        ),
        5000000
      )

      const { remoteReceipt } = await env.waitForXDomainTransaction(
        transaction,
        Direction.L1ToL2
      )

      expect(remoteReceipt.status).to.equal(0)
    })

    it('should have a receipt with a status of 0 for messages sent to 0xdead... addresses', async () => {
      const transaction = await env.l1Messenger.sendMessage(
        '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',
        '0x',
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
