import { expect } from 'chai'

/* Imports: External */
import { Contract, ContractFactory, Wallet, utils } from 'ethers'

/* Imports: Internal */
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
import simpleStorageJson from '../artifacts/contracts/SimpleStorage.sol/SimpleStorage.json'
import { fundUser, isLiveNetwork } from './shared/utils'

// Need a big timeout to allow for all transactions to be processed.
// For some reason I can't figure out how to set the timeout on a per-suite basis
// so I'm instead setting it for every test.
const STRESS_TEST_TIMEOUT = isLiveNetwork() ? 500_000 : 1_200_000

describe('stress tests', () => {
  const numTransactions = 3

  let env: OptimismEnv

  const wallets: Wallet[] = []

  before(async () => {
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
    const factory__L1SimpleStorage = new ContractFactory(
      simpleStorageJson.abi,
      simpleStorageJson.bytecode,
      env.l1Wallet
    )
    const factory__L2SimpleStorage = new ContractFactory(
      simpleStorageJson.abi,
      simpleStorageJson.bytecode,
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
})
