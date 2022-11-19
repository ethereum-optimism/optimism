/* Imports: External */
import { Contract, ContractFactory } from 'ethers'
import { ethers } from 'hardhat'
import { MessageDirection, MessageStatus } from '@eth-optimism/sdk'
import {
  applyL1ToL2Alias,
  awaitCondition,
  sleep,
} from '@eth-optimism/core-utils'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import {
  DEFAULT_TEST_GAS_L1,
  DEFAULT_TEST_GAS_L2,
  envConfig,
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
        const transaction = await env.messenger.sendMessage(
          {
            direction: MessageDirection.L2_TO_L1,
            target: L1SimpleStorage.address,
            message: L1SimpleStorage.interface.encodeFunctionData('setValue', [
              value,
            ]),
          },
          {
            overrides: {
              gasLimit: DEFAULT_TEST_GAS_L2,
            },
          }
        )

        await env.messenger.waitForMessageStatus(
          transaction,
          MessageStatus.READY_FOR_RELAY
        )

        await env.messenger.finalizeMessage(transaction)
        await env.messenger.waitForMessageReceipt(transaction)

        expect(await L1SimpleStorage.msgSender()).to.equal(
          env.messenger.contracts.l1.L1CrossDomainMessenger.address
        )
        expect(await L1SimpleStorage.xDomainSender()).to.equal(
          await env.messenger.l2Signer.getAddress()
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
      const transaction = await env.messenger.sendMessage(
        {
          direction: MessageDirection.L1_TO_L2,
          target: L2SimpleStorage.address,
          message: L2SimpleStorage.interface.encodeFunctionData('setValue', [
            value,
          ]),
        },
        {
          l2GasLimit: 5000000,
          overrides: {
            gasLimit: DEFAULT_TEST_GAS_L1,
          },
        }
      )

      const receipt = await env.messenger.waitForMessageReceipt(transaction)

      expect(receipt.transactionReceipt.status).to.equal(1)
      expect(await L2SimpleStorage.msgSender()).to.equal(
        env.messenger.contracts.l2.L2CrossDomainMessenger.address
      )
      expect(await L2SimpleStorage.txOrigin()).to.equal(
        applyL1ToL2Alias(
          env.messenger.contracts.l1.L1CrossDomainMessenger.address
        )
      )
      expect(await L2SimpleStorage.xDomainSender()).to.equal(
        await env.messenger.l1Signer.getAddress()
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
      const tx =
        await env.messenger.contracts.l1.CanonicalTransactionChain.connect(
          env.messenger.l1Signer
        ).enqueue(
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
      let currBlock = await env.messenger.l1Provider.getBlockNumber()
      while (currBlock <= waitUntilBlock) {
        const progress =
          envConfig.DTL_ENQUEUE_CONFIRMATIONS - (waitUntilBlock - currBlock)
        console.log(
          `Waiting for ${progress}/${envConfig.DTL_ENQUEUE_CONFIRMATIONS} confirmations.`
        )
        await sleep(5000)
        currBlock = await env.messenger.l1Provider.getBlockNumber()
      }
      console.log('Enqueue should be confirmed.')

      await awaitCondition(
        async () => {
          const sender = await L2SimpleStorage.msgSender()
          return sender === (await env.messenger.l1Signer.getAddress())
        },
        2000,
        60
      )

      // No aliasing when an EOA goes directly to L2.
      expect(await L2SimpleStorage.msgSender()).to.equal(
        await env.messenger.l1Signer.getAddress()
      )
      expect(await L2SimpleStorage.txOrigin()).to.equal(
        await env.messenger.l1Signer.getAddress()
      )
      expect(await L2SimpleStorage.value()).to.equal(value)
      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(1)
    })
  })
})
