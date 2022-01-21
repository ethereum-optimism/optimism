/* Imports: External */
import { Contract, ContractFactory } from 'ethers'
import { ethers } from 'hardhat'
import { applyL1ToL2Alias, awaitCondition } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { expect } from './shared/setup'
import { Direction } from './shared/watcher-utils'
import { OptimismEnv } from './shared/env'
import {
  DEFAULT_TEST_GAS_L1,
  DEFAULT_TEST_GAS_L2,
  envConfig,
  sleep,
  withdrawalTest,
} from './shared/utils'

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
    Factory__L1SimpleStorage = await ethers.getContractFactory(
      'SimpleStorage',
      env.l1Wallet
    )
    Factory__L2SimpleStorage = await ethers.getContractFactory(
      'SimpleStorage',
      env.l2Wallet
    )
    Factory__L2Reverter = await ethers.getContractFactory(
      'Reverter',
      env.l2Wallet
    )
  })

  beforeEach(async () => {
    L1SimpleStorage = await Factory__L1SimpleStorage.deploy()
    await L1SimpleStorage.deployed()
    L2SimpleStorage = await Factory__L2SimpleStorage.deploy()
    await L2SimpleStorage.deployed()
    L2Reverter = await Factory__L2Reverter.deploy()
    await L2Reverter.deployed()
  })

  describe('L2 => L1', () => {
    withdrawalTest(
      'should be able to perform a withdrawal from L2 -> L1',
      async () => {
        const value = `0x${'77'.repeat(32)}`

        // Send L2 -> L1 message.
        const transaction = await env.l2Messenger.sendMessage(
          L1SimpleStorage.address,
          L1SimpleStorage.interface.encodeFunctionData('setValue', [value]),
          5000000,
          {
            gasLimit: DEFAULT_TEST_GAS_L2,
          }
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
      }
    )
  })

  describe('L1 => L2', () => {
    it('should deposit from L1 -> L2', async () => {
      const value = `0x${'42'.repeat(32)}`

      // Send L1 -> L2 message.
      const transaction = await env.l1Messenger.sendMessage(
        L2SimpleStorage.address,
        L2SimpleStorage.interface.encodeFunctionData('setValue', [value]),
        5000000,
        {
          gasLimit: DEFAULT_TEST_GAS_L1,
        }
      )

      await env.waitForXDomainTransaction(transaction, Direction.L1ToL2)

      expect(await L2SimpleStorage.msgSender()).to.equal(
        env.l2Messenger.address
      )
      expect(await L2SimpleStorage.txOrigin()).to.equal(
        applyL1ToL2Alias(env.l1Messenger.address)
      )
      expect(await L2SimpleStorage.xDomainSender()).to.equal(
        env.l1Wallet.address
      )
      expect(await L2SimpleStorage.value()).to.equal(value)
      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(1)
    })

    it('should deposit from L1 -> L2 directly via enqueue', async function () {
      this.timeout(
        envConfig.MOCHA_TIMEOUT * 2 +
          envConfig.DTL_ENQUEUE_CONFIRMATIONS * 15000
      )
      const value = `0x${'42'.repeat(32)}`

      // Send L1 -> L2 message.
      const tx = await env.ctc
        .connect(env.l1Wallet)
        .enqueue(
          L2SimpleStorage.address,
          5000000,
          L2SimpleStorage.interface.encodeFunctionData('setValueNotXDomain', [
            value,
          ]),
          {
            gasLimit: DEFAULT_TEST_GAS_L1,
          }
        )
      const receipt = await tx.wait()

      const waitUntilBlock =
        receipt.blockNumber + envConfig.DTL_ENQUEUE_CONFIRMATIONS
      let currBlock = await env.l1Provider.getBlockNumber()
      while (currBlock <= waitUntilBlock) {
        const progress =
          envConfig.DTL_ENQUEUE_CONFIRMATIONS - (waitUntilBlock - currBlock)
        console.log(
          `Waiting for ${progress}/${envConfig.DTL_ENQUEUE_CONFIRMATIONS} confirmations.`
        )
        await sleep(5000)
        currBlock = await env.l1Provider.getBlockNumber()
      }
      console.log('Enqueue should be confirmed.')

      await awaitCondition(
        async () => {
          const sender = await L2SimpleStorage.msgSender()
          return sender === env.l1Wallet.address
        },
        2000,
        60
      )

      // No aliasing when an EOA goes directly to L2.
      expect(await L2SimpleStorage.msgSender()).to.equal(env.l1Wallet.address)
      expect(await L2SimpleStorage.txOrigin()).to.equal(env.l1Wallet.address)
      expect(await L2SimpleStorage.value()).to.equal(value)
      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(1)
    })

    it('should have a receipt with a status of 1 for a successful message', async () => {
      const value = `0x${'42'.repeat(32)}`

      // Send L1 -> L2 message.
      const transaction = await env.l1Messenger.sendMessage(
        L2SimpleStorage.address,
        L2SimpleStorage.interface.encodeFunctionData('setValue', [value]),
        5000000,
        {
          gasLimit: DEFAULT_TEST_GAS_L1,
        }
      )
      await transaction.wait()

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
        5000000,
        {
          gasLimit: DEFAULT_TEST_GAS_L1,
        }
      )

      const { remoteReceipt } = await env.waitForXDomainTransaction(
        transaction,
        Direction.L1ToL2
      )

      expect(remoteReceipt.status).to.equal(0)
    })
  })
})
