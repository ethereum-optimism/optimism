import '../setup'

/* External Imports */
import { PostgresDB, Row } from '@eth-optimism/core-db'
import {keccak256FromUtf8} from '@eth-optimism/core-utils'

/* Internal Imports */
import { DefaultDataService } from '../../src/app/data'
import {QueueOrigin} from '../../src/types/data'
import {createRollupTx, createTx, l1Block, verifyL1BlockRes, verifyL1RollupTx, verifyL1TxRes} from './helpers'


describe('Data Service (will fail if postgres is not running with expected schema)', () => {
  let dataService: DefaultDataService
  let postgres: PostgresDB
  before(async () => {
    postgres = new PostgresDB('0.0.0.0', 5432, 'test', 'test', 'rollup')
    dataService = new DefaultDataService(postgres)
  })

  beforeEach(async () => {
    await postgres.execute(`DELETE FROM l2_tx_output`)
    await postgres.execute(`DELETE FROM state_commitment_chain_batch`)
    await postgres.execute(`DELETE FROM canonical_chain_batch`)
    await postgres.execute(`DELETE FROM l1_rollup_tx`)
    await postgres.execute(`DELETE FROM l1_rollup_state_root`)
    await postgres.execute(`DELETE FROM l1_rollup_state_root_batch`)
    await postgres.execute(`DELETE FROM geth_submission_queue`)
    await postgres.execute(`DELETE FROM l1_tx`)
    await postgres.execute(`DELETE FROM l1_block`)
  })

  describe('insertL1Block', () => {
    it('Should insert an L1 block', async () => {
      await dataService.insertL1Block(l1Block)

      const res = await postgres.select(`SELECT * FROM l1_block`)
      res.length.should.equal(1, `No L1 Block rows!`)
      verifyL1BlockRes(res[0], l1Block, false)
    })

    it('Should insert a processed L1 block', async () => {
      await dataService.insertL1Block(l1Block, true)

      const res = await postgres.select(`SELECT * FROM l1_block`)
      verifyL1BlockRes(res[0], l1Block, true)
    })
  })

  describe('insertL1Transactions', () => {
    it('Should insert a L1 transactions', async () => {
      await dataService.insertL1Block(l1Block)

      const tx1 = createTx(keccak256FromUtf8('tx 1'))
      const tx2 = createTx(keccak256FromUtf8('tx 2'))
      await dataService.insertL1Transactions([tx1, tx2])

      const res = await postgres.select(
        `SELECT * FROM l1_tx ORDER BY tx_index ASC`
      )
      res.length.should.equal(2, `No L1 Tx rows!`)
      verifyL1TxRes(res[0], tx1, 0)
      verifyL1TxRes(res[1], tx2, 1)
    })
  })

  describe('insertL1BlockAndTransactions', () => {
    it('Should insert an L1 block and transactions', async () => {
      const tx1 = createTx(keccak256FromUtf8('tx 1'))
      const tx2 = createTx(keccak256FromUtf8('tx 2'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx1, tx2])
      const blockRes = await postgres.select(`SELECT * FROM l1_block`)
      blockRes.length.should.equal(1, `No L1 Block rows!`)
      verifyL1BlockRes(blockRes[0], l1Block, false)

      const txRes = await postgres.select(
        `SELECT * FROM l1_tx ORDER BY tx_index ASC`
      )
      txRes.length.should.equal(2, `No L1 Tx rows!`)
      verifyL1TxRes(txRes[0], tx1, 0)
      verifyL1TxRes(txRes[1], tx2, 1)
    })

    it('Should insert an L1 block and transactions (processed)', async () => {
      const tx1 = createTx(keccak256FromUtf8('tx 1'))
      const tx2 = createTx(keccak256FromUtf8('tx 2'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx1, tx2], true)
      const blockRes = await postgres.select(`SELECT * FROM l1_block`)
      blockRes.length.should.equal(1, `No L1 Block rows!`)
      verifyL1BlockRes(blockRes[0], l1Block, true)

      const txRes = await postgres.select(
        `SELECT * FROM l1_tx ORDER BY tx_index ASC`
      )
      txRes.length.should.equal(2, `No L1 Tx rows!`)
      verifyL1TxRes(txRes[0], tx1, 0)
      verifyL1TxRes(txRes[1], tx2, 1)
    })
  })

  describe('insertL1RollupTransactions', () => {
    it('Should insert rollup transactions without queueing geth submission', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx])

      const rTx1 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE)
      const rTx2 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE, 0, 1)

      const submissionIndex = await dataService.insertL1RollupTransactions(tx.hash, [rTx1, rTx2])

      const indexExists = submissionIndex !== undefined
      indexExists.should.equal(false, `Geth submission should not be scheduled!`)

      const res: Row[] = await postgres.select(`SELECT * FROM l1_rollup_tx ORDER BY l1_tx_index ASC, l1_tx_log_index ASC, index_within_submission ASC `)
      res.length.should.equal(2, `Incorrect # of Rollup Tx entries!`)
      verifyL1RollupTx(res[0], rTx1)
      verifyL1RollupTx(res[1], rTx2)

      const submissionRes = await postgres.select(`SELECT * FROM geth_submission_queue`)
      submissionRes.length.should.equal(0, `Geth submission queued!`)
    })

    it('Should insert rollup transactions queueing geth submission', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx])

      const rTx1 = createRollupTx(tx, QueueOrigin.SEQUENCER)
      const rTx2 = createRollupTx(tx, QueueOrigin.SEQUENCER, 0, 1)

      const submissionIndex = await dataService.insertL1RollupTransactions(tx.hash, [rTx1, rTx2], true)

      const indexExists = submissionIndex !== undefined
      indexExists.should.equal(true, `Geth submission should not be scheduled!`)

      const res: Row[] = await postgres.select(`SELECT * FROM l1_rollup_tx ORDER BY l1_tx_index ASC, l1_tx_log_index ASC, index_within_submission ASC `)
      res.length.should.equal(2, `Incorrect # of Rollup Tx entries!`)
      verifyL1RollupTx(res[0], rTx1)
      verifyL1RollupTx(res[1], rTx2)

      const submissionRes = await postgres.select(`SELECT * FROM geth_submission_queue`)
      submissionRes.length.should.equal(1, `Geth submission queued!`)
    })
  })
})
