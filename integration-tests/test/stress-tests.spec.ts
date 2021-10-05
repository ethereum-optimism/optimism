import { expect } from 'chai'

/* Imports: External */
import { Contract, ContractFactory } from 'ethers'

/* Imports: Internal */
import { OptimismEnv } from './shared/env'
import {
  executeRepeatedL1ToL2Transactions,
  executeRepeatedL2ToL1Transactions,
  executeRepeatedL2Transactions,
  executeRepeatedL1ToL2TransactionsParallel,
  executeRepeatedL2ToL1TransactionsParallel,
  executeRepeatedL2TransactionsParallel,
} from './shared/stress-test-helpers'

/* Imports: Artifacts */
import simpleStorageJson from '../artifacts/contracts/SimpleStorage.sol/SimpleStorage.json'

// Need a big timeout to allow for all transactions to be processed.
// For some reason I can't figure out how to set the timeout on a per-suite basis
// so I'm instead setting it for every test.
const STRESS_TEST_TIMEOUT = 500_000

describe('stress tests', () => {
  let env: OptimismEnv
  before(async () => {
    env = await OptimismEnv.new()
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
    const numTransactions = 10

    it(`${numTransactions} L1 => L2 transactions (serial)`, async () => {
      await executeRepeatedL1ToL2Transactions(
        env,
        {
          contract: L2SimpleStorage,
          functionName: 'setValue',
          functionParams: [`0x${'42'.repeat(32)}`],
        },
        numTransactions
      )

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions
      )
    }).timeout(STRESS_TEST_TIMEOUT)

    it(`${numTransactions} L1 => L2 transactions (parallel)`, async () => {
      await executeRepeatedL1ToL2TransactionsParallel(
        env,
        {
          contract: L2SimpleStorage,
          functionName: 'setValue',
          functionParams: [`0x${'42'.repeat(32)}`],
        },
        numTransactions
      )

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions
      )
    }).timeout(STRESS_TEST_TIMEOUT)
  })

  describe('L2 => L1 stress tests', () => {
    const numTransactions = 10

    it(`${numTransactions} L2 => L1 transactions (serial)`, async () => {
      await executeRepeatedL2ToL1Transactions(
        env,
        {
          contract: L1SimpleStorage,
          functionName: 'setValue',
          functionParams: [`0x${'42'.repeat(32)}`],
        },
        numTransactions
      )

      expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions
      )
    }).timeout(STRESS_TEST_TIMEOUT)

    it(`${numTransactions} L2 => L1 transactions (parallel)`, async () => {
      await executeRepeatedL2ToL1TransactionsParallel(
        env,
        {
          contract: L1SimpleStorage,
          functionName: 'setValue',
          functionParams: [`0x${'42'.repeat(32)}`],
        },
        numTransactions
      )

      expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions
      )
    }).timeout(STRESS_TEST_TIMEOUT)
  })

  describe('L2 transaction stress tests', () => {
    const numTransactions = 10

    it(`${numTransactions} L2 transactions (serial)`, async () => {
      await executeRepeatedL2Transactions(
        env,
        {
          contract: L2SimpleStorage,
          functionName: 'setValueNotXDomain',
          functionParams: [`0x${'42'.repeat(32)}`],
        },
        numTransactions
      )

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions
      )
    }).timeout(STRESS_TEST_TIMEOUT)

    it(`${numTransactions} L2 transactions (parallel)`, async () => {
      await executeRepeatedL2TransactionsParallel(
        env,
        {
          contract: L2SimpleStorage,
          functionName: 'setValueNotXDomain',
          functionParams: [`0x${'42'.repeat(32)}`],
        },
        numTransactions
      )

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions
      )
    }).timeout(STRESS_TEST_TIMEOUT)
  })

  describe('C-C-C-Combo breakers', () => {
    const numTransactions = 10

    it(`${numTransactions} L2 transactions, L1 => L2 transactions, L2 => L1 transactions (txs serial, suites parallel)`, async () => {
      await Promise.all([
        executeRepeatedL1ToL2Transactions(
          env,
          {
            contract: L2SimpleStorage,
            functionName: 'setValue',
            functionParams: [`0x${'42'.repeat(32)}`],
          },
          numTransactions
        ),
        executeRepeatedL2ToL1Transactions(
          env,
          {
            contract: L1SimpleStorage,
            functionName: 'setValue',
            functionParams: [`0x${'42'.repeat(32)}`],
          },
          numTransactions
        ),
        executeRepeatedL2Transactions(
          env,
          {
            contract: L2SimpleStorage,
            functionName: 'setValueNotXDomain',
            functionParams: [`0x${'42'.repeat(32)}`],
          },
          numTransactions
        ),
      ])

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions * 2
      )

      expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions
      )
    }).timeout(STRESS_TEST_TIMEOUT)

    it(`${numTransactions} L2 transactions, L1 => L2 transactions, L2 => L1 transactions (all parallel)`, async () => {
      await Promise.all([
        executeRepeatedL1ToL2TransactionsParallel(
          env,
          {
            contract: L2SimpleStorage,
            functionName: 'setValue',
            functionParams: [`0x${'42'.repeat(32)}`],
          },
          numTransactions
        ),
        executeRepeatedL2ToL1TransactionsParallel(
          env,
          {
            contract: L1SimpleStorage,
            functionName: 'setValue',
            functionParams: [`0x${'42'.repeat(32)}`],
          },
          numTransactions
        ),
        executeRepeatedL2TransactionsParallel(
          env,
          {
            contract: L2SimpleStorage,
            functionName: 'setValueNotXDomain',
            functionParams: [`0x${'42'.repeat(32)}`],
          },
          numTransactions
        ),
      ])

      expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions * 2
      )

      expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(
        numTransactions
      )
    }).timeout(STRESS_TEST_TIMEOUT)
  })
})
