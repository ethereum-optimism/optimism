import '../setup'

/* External Imports */
import { PostgresDB, Row } from '@eth-optimism/core-db'
import { keccak256FromUtf8 } from '@eth-optimism/core-utils'

/* Internal Imports */
import { DefaultDataService } from '../../src/app/data'
import {
  GethSubmissionQueueStatus,
  QueueOrigin,
  VerificationStatus,
} from '../../src/types/data'
import {
  createRollupTx,
  createTx,
  deleteAllData,
  l1Block,
  verifyL1BlockRes,
  verifyL1RollupTx,
  verifyL1TxRes,
  verifyStateRoot,
} from './helpers'

describe('L1 Data Service (will fail if postgres is not running with expected schema)', () => {
  let dataService: DefaultDataService
  let postgres: PostgresDB
  before(async () => {
    postgres = new PostgresDB('0.0.0.0', 5432, 'test', 'test', 'rollup')
    dataService = new DefaultDataService(postgres)
  })

  beforeEach(async () => {
    await deleteAllData(postgres)
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

    it('Should update an inserted block to processed', async () => {
      await dataService.insertL1Block(l1Block, false)

      let res = await postgres.select(`SELECT * FROM l1_block`)
      verifyL1BlockRes(res[0], l1Block, false)

      await dataService.updateBlockToProcessed(l1Block.hash)

      res = await postgres.select(`SELECT * FROM l1_block`)
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

      const submissionIndex = await dataService.insertL1RollupTransactions(
        tx.hash,
        [rTx1, rTx2]
      )
      submissionIndex.should.equal(
        -1,
        `Geth submission should not be scheduled!`
      )

      const res: Row[] = await postgres.select(
        `SELECT * FROM l1_rollup_tx ORDER BY l1_tx_index ASC, l1_tx_log_index ASC, index_within_submission ASC `
      )
      res.length.should.equal(2, `Incorrect # of Rollup Tx entries!`)
      verifyL1RollupTx(res[0], rTx1)
      verifyL1RollupTx(res[1], rTx2)

      const submissionRes = await postgres.select(
        `SELECT * FROM geth_submission_queue`
      )
      submissionRes.length.should.equal(0, `Geth submission queued!`)
    })

    it('Should insert rollup transactions queueing geth submission', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx])

      const rTx1 = createRollupTx(tx, QueueOrigin.SEQUENCER)
      const rTx2 = createRollupTx(tx, QueueOrigin.SEQUENCER, 0, 1)

      const submissionIndex = await dataService.insertL1RollupTransactions(
        tx.hash,
        [rTx1, rTx2],
        true
      )
      submissionIndex.should.equal(0, `Geth submission should be scheduled!`)

      const res: Row[] = await postgres.select(
        `SELECT * FROM l1_rollup_tx ORDER BY l1_tx_index ASC, l1_tx_log_index ASC, index_within_submission ASC `
      )
      res.length.should.equal(2, `Incorrect # of Rollup Tx entries!`)
      verifyL1RollupTx(res[0], rTx1)
      verifyL1RollupTx(res[1], rTx2)

      const submissionRes = await postgres.select(
        `SELECT * FROM geth_submission_queue`
      )
      submissionRes.length.should.equal(1, `Geth submission queued!`)
    })
  })

  describe('queueNextGethSubmission', () => {
    it('Should queue unqueued geth submission matching queue origin', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx], true)

      const rTx1 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE)
      const rTx2 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE, 0, 1)

      let submissionIndex = await dataService.insertL1RollupTransactions(
        tx.hash,
        [rTx1, rTx2]
      )
      submissionIndex.should.equal(
        -1,
        `Geth submission should not be scheduled!`
      )

      let submissionRes = await postgres.select(
        `SELECT status FROM geth_submission_queue`
      )
      submissionRes.length.should.equal(0, `Geth submission queued!`)

      submissionIndex = await dataService.queueNextGethSubmission([
        QueueOrigin.SAFETY_QUEUE,
      ])
      submissionIndex.should.equal(0, `Geth submission should be scheduled!`)

      submissionRes = await postgres.select(
        `SELECT status FROM geth_submission_queue`
      )
      submissionRes.length.should.equal(1, `Geth submission not queued!`)
      submissionRes[0]['status'].should.equal(
        GethSubmissionQueueStatus.QUEUED,
        `Incorrect queue status!`
      )
    })

    it('Should not queue unqueued geth submission if l1 block is not processed', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx], false)

      const rTx1 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE)
      const rTx2 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE, 0, 1)

      let submissionIndex = await dataService.insertL1RollupTransactions(
        tx.hash,
        [rTx1, rTx2]
      )
      submissionIndex.should.equal(
        -1,
        `Geth submission should not be scheduled!`
      )

      let submissionRes = await postgres.select(
        `SELECT status FROM geth_submission_queue`
      )
      submissionRes.length.should.equal(0, `Geth submission queued!`)

      submissionIndex = await dataService.queueNextGethSubmission([
        QueueOrigin.SAFETY_QUEUE,
      ])
      submissionIndex.should.equal(
        -1,
        `Geth submission should not be scheduled!`
      )

      submissionRes = await postgres.select(
        `SELECT status FROM geth_submission_queue`
      )
      submissionRes.length.should.equal(0, `Geth submission queued!`)
    })

    it('Should not queue unqueued geth submission if wrong queue origin', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx], true)

      const rTx1 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE)
      const rTx2 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE, 0, 1)

      let submissionIndex = await dataService.insertL1RollupTransactions(
        tx.hash,
        [rTx1, rTx2]
      )
      submissionIndex.should.equal(
        -1,
        `Geth submission should not be scheduled!`
      )

      let submissionRes = await postgres.select(
        `SELECT status FROM geth_submission_queue`
      )
      submissionRes.length.should.equal(0, `Geth submission queued!`)

      submissionIndex = await dataService.queueNextGethSubmission([
        QueueOrigin.SEQUENCER,
        QueueOrigin.L1_TO_L2_QUEUE,
      ])
      submissionIndex.should.equal(
        -1,
        `Geth submission should not be scheduled!`
      )

      submissionRes = await postgres.select(
        `SELECT status FROM geth_submission_queue`
      )
      submissionRes.length.should.equal(0, `Geth submission queued!`)
    })
  })

  describe('insertL1RollupStateRoots', () => {
    it('No state root batch should exist by default', async () => {
      const batchRes = await postgres.select(
        `SELECT status FROM l1_rollup_state_root_batch`
      )
      batchRes.length.should.equal(0, `No rollup batches should exist`)
    })

    it('Should insert state root batch', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx], true)

      const stateRoots = [
        keccak256FromUtf8('root1'),
        keccak256FromUtf8('root2'),
        keccak256FromUtf8('root3'),
      ]

      const batchNumber = await dataService.insertL1RollupStateRoots(
        tx.hash,
        stateRoots
      )
      const rootRes = await postgres.select(
        `SELECT * FROM l1_rollup_state_root`
      )
      rootRes.length.should.equal(3, `State roots not inserted!`)
      verifyStateRoot(rootRes[0], stateRoots[0], 0, batchNumber)
      verifyStateRoot(rootRes[1], stateRoots[1], 1, batchNumber)
      verifyStateRoot(rootRes[2], stateRoots[2], 2, batchNumber)

      batchNumber.should.equal(0, `State root should be batched!`)

      const batchRes = await postgres.select(
        `SELECT status FROM l1_rollup_state_root_batch`
      )
      batchRes.length.should.equal(1, `State root batch should exist!`)
      batchRes[0]['status'].should.equal(
        VerificationStatus.UNVERIFIED,
        `Incorrect status!`
      )
    })
  })

  describe('getNextQueuedGethSubmission', () => {
    it('Should be empty without any queued geth submissions', async () => {
      const res = await postgres.select(
        `SELECT * FROM next_queued_geth_submission`
      )
      res.length.should.equal(0, `No queued geth submissions should exist!`)
    })

    it('Should pick up queued batch submission', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [tx], true)

      const rTx1 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE)
      const rTx2 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE, 0, 1)

      const submissionIndex = await dataService.insertL1RollupTransactions(
        tx.hash,
        [rTx1, rTx2],
        true
      )
      submissionIndex.should.equal(0, `Geth submission should be scheduled!`)

      const indexRes = await postgres.select(
        `SELECT geth_submission_queue_index FROM next_queued_geth_submission`
      )
      indexRes.length.should.equal(2, `Result should have 2 txs!`)
      indexRes[0]['geth_submission_queue_index'].should.equal(
        submissionIndex.toString(10),
        `Incorrect submission index!`
      )
    })
  })

  describe('markQueuedGethSubmissionSubmittedToGeth', () => {
    it('Should properly update queued submission', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [tx], true)

      const rTx1 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE)
      const rTx2 = createRollupTx(tx, QueueOrigin.SAFETY_QUEUE, 0, 1)

      const submissionIndex: number = await dataService.insertL1RollupTransactions(
        tx.hash,
        [rTx1, rTx2],
        true
      )
      submissionIndex.should.equal(0, `Geth submission should be scheduled!`)

      let indexRes = await postgres.select(
        `SELECT geth_submission_queue_index FROM next_queued_geth_submission`
      )
      indexRes.length.should.equal(2, `Result should have 2 txs!`)
      indexRes[0]['geth_submission_queue_index'].should.equal(
        submissionIndex.toString(10),
        `Incorrect submission index!`
      )

      await dataService.markQueuedGethSubmissionSubmittedToGeth(submissionIndex)

      indexRes = await postgres.select(
        `SELECT geth_submission_queue_index FROM next_queued_geth_submission`
      )
      indexRes.length.should.equal(
        0,
        `Result should have 0 tx because they are submitted!`
      )

      const queueRes = await postgres.select(
        `SELECT status FROM geth_submission_queue WHERE queue_index = ${submissionIndex}`
      )
      queueRes.length.should.equal(1, `Should have queue record!`)
      queueRes[0]['status'].should.equal(
        GethSubmissionQueueStatus.SENT,
        `Not updated to sent!`
      )
    })
  })
})
