/* Imports: External */
import { Contract, Wallet, utils } from 'ethers'
import { ethers } from 'hardhat'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import {
  executeL1ToL2TransactionsParallel,
  executeL2ToL1TransactionsParallel,
  executeL2TransactionsParallel,
  executeRepeatedL1ToL2Transactions,
  executeRepeatedL2ToL1Transactions,
  executeRepeatedL2Transactions,
  fundRandomWallet,
} from './shared/stress-test-helpers'
/* Imports: Artifacts */
import { envConfig, fundUser } from './shared/utils'

// Need a big timeout to allow for all transactions to be processed.
// For some reason I can't figure out how to set the timeout on a per-suite basis
// so I'm instead setting it for every test.
const STRESS_TEST_TIMEOUT = envConfig.MOCHA_TIMEOUT * 5

describe('stress tests', () => {
  const numTransactions = 3

  let env: OptimismEnv

  const wallets: Wallet[] = []

  before(async function () {
    if (!envConfig.RUN_STRESS_TESTS) {
      console.log('Skipping stress tests.')
      this.skip()
      return
    }

    env = await OptimismEnv.new()

    for (let i = 0; i < numTransactions; i++) {
      wallets.push(Wallet.createRandom())
    }

    for (const wallet of wallets) {
      await fundRandomWallet(env, wallet, utils.parseEther('0.1'))
    }

    for (const wallet of wallets) {
      await fundUser(
        env.watcher,
        env.l1Bridge,
        utils.parseEther('0.1'),
        wallet.address
      )
    }
  })

  let L2SimpleStorage: Contract
  let L1SimpleStorage: Contract
  beforeEach(async () => {
    const factory__L1SimpleStorage = await ethers.getContractFactory(
      'SimpleStorage',
      env.l1Wallet
    )
    const factory__L2SimpleStorage = await ethers.getContractFactory(
      'SimpleStorage',
      env.l2Wallet
    )
    L1SimpleStorage = await factory__L1SimpleStorage.deploy()
    await L1SimpleStorage.deployTransaction.wait()
    L2SimpleStorage = await factory__L2SimpleStorage.deploy()
    await L2SimpleStorage.deployTransaction.wait()
  })

  describe('L1 => L2 stress tests', () => {
    it(`${numTransactions} L1 => L2 transactions (serial)`, async () => {
      await executeRepeatedL1ToL2Transactions(env, wallets, {
        contract: L2SimpleStorage,
        functionName: 'setValue',
        functionParams: [`0x${'42'.repeat(32)}`],
      })

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        wallets.length
      )
    }).timeout(STRESS_TEST_TIMEOUT)

    it(`${numTransactions} L1 => L2 transactions (parallel)`, async () => {
      await executeL1ToL2TransactionsParallel(env, wallets, {
        contract: L2SimpleStorage,
        functionName: 'setValue',
        functionParams: [`0x${'42'.repeat(32)}`],
      })

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        wallets.length
      )
    }).timeout(STRESS_TEST_TIMEOUT)
  })

  describe('L2 => L1 stress tests', () => {
    it(`${numTransactions} L2 => L1 transactions (serial)`, async () => {
      await executeRepeatedL2ToL1Transactions(env, wallets, {
        contract: L1SimpleStorage,
        functionName: 'setValue',
        functionParams: [`0x${'42'.repeat(32)}`],
      })

      expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(
        wallets.length
      )
    }).timeout(STRESS_TEST_TIMEOUT)

    it(`${numTransactions} L2 => L1 transactions (parallel)`, async () => {
      await executeL2ToL1TransactionsParallel(env, wallets, {
        contract: L1SimpleStorage,
        functionName: 'setValue',
        functionParams: [`0x${'42'.repeat(32)}`],
      })

      expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(
        wallets.length
      )
    }).timeout(STRESS_TEST_TIMEOUT)
  })

  describe('L2 transaction stress tests', () => {
    it(`${numTransactions} L2 transactions (serial)`, async () => {
      await executeRepeatedL2Transactions(env, wallets, {
        contract: L2SimpleStorage,
        functionName: 'setValueNotXDomain',
        functionParams: [`0x${'42'.repeat(32)}`],
      })

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        wallets.length
      )
    }).timeout(STRESS_TEST_TIMEOUT)

    it(`${numTransactions} L2 transactions (parallel)`, async () => {
      await executeL2TransactionsParallel(env, wallets, {
        contract: L2SimpleStorage,
        functionName: 'setValueNotXDomain',
        functionParams: [`0x${'42'.repeat(32)}`],
      })

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions
      )
    }).timeout(STRESS_TEST_TIMEOUT)
  })

  describe('C-C-C-Combo breakers', () => {
    it(`${numTransactions} L2 transactions, L1 => L2 transactions, L2 => L1 transactions (txs serial, suites parallel)`, async () => {
      await Promise.all([
        executeRepeatedL1ToL2Transactions(env, wallets, {
          contract: L2SimpleStorage,
          functionName: 'setValue',
          functionParams: [`0x${'42'.repeat(32)}`],
        }),
        executeRepeatedL2ToL1Transactions(env, wallets, {
          contract: L1SimpleStorage,
          functionName: 'setValue',
          functionParams: [`0x${'42'.repeat(32)}`],
        }),
        executeRepeatedL2Transactions(env, wallets, {
          contract: L2SimpleStorage,
          functionName: 'setValueNotXDomain',
          functionParams: [`0x${'42'.repeat(32)}`],
        }),
      ])

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        wallets.length * 2
      )

      expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(
        wallets.length
      )
    }).timeout(STRESS_TEST_TIMEOUT)

    it(`${numTransactions} L2 transactions, L1 => L2 transactions, L2 => L1 transactions (all parallel)`, async () => {
      await Promise.all([
        executeL1ToL2TransactionsParallel(env, wallets, {
          contract: L2SimpleStorage,
          functionName: 'setValue',
          functionParams: [`0x${'42'.repeat(32)}`],
        }),
        executeL2ToL1TransactionsParallel(env, wallets, {
          contract: L1SimpleStorage,
          functionName: 'setValue',
          functionParams: [`0x${'42'.repeat(32)}`],
        }),
        executeL2TransactionsParallel(env, wallets, {
          contract: L2SimpleStorage,
          functionName: 'setValueNotXDomain',
          functionParams: [`0x${'42'.repeat(32)}`],
        }),
      ])

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        wallets.length * 2
      )

      expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(
        wallets.length
      )
    }).timeout(STRESS_TEST_TIMEOUT)
  })

  // These tests depend on an archive node due to the historical `eth_call`s
  describe('Monotonicity Checks', () => {
    it('should have monotonic timestamps and l1 blocknumbers', async () => {
      const tip = await env.l2Provider.getBlock('latest')
      const prev = {
        block: await env.l2Provider.getBlock(0),
        l1BlockNumber: await env.l1BlockNumber.getL1BlockNumber({
          blockTag: 0,
        }),
      }
      for (let i = 1; i < tip.number; i++) {
        const block = await env.l2Provider.getBlock(i)
        expect(block.timestamp).to.be.gte(prev.block.timestamp)

        const l1BlockNumber = await env.l1BlockNumber.getL1BlockNumber({
          blockTag: i,
        })
        expect(l1BlockNumber.gt(prev.l1BlockNumber))

        prev.block = block
        prev.l1BlockNumber = l1BlockNumber
      }
    })
  })
})
