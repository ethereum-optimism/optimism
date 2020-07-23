import '../setup'

/* External Imports */
import { PostgresDB, Row } from '@eth-optimism/core-db'
import { keccak256FromUtf8 } from '@eth-optimism/core-utils'

/* Internal Imports */
import { DefaultDataService } from '../../src/app/data'
import {
  blockNumber,
  createTxOutput,
  defaultStateRoot,
  l1Block,
  verifyL1BlockRes,
  verifyL2TxOutput,
} from './helpers'
import { BatchSubmissionStatus } from '../../src/types/data'

describe('L2 Data Service (will fail if postgres is not running with expected schema)', () => {
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

  describe('insertL2TransactionOutput', () => {
    it('Should insert L2 Tx Output', async () => {
      const tx = createTxOutput(keccak256FromUtf8('tx'))
      await dataService.insertL2TransactionOutput(tx)

      const res = await postgres.select(`SELECT * FROM l2_tx_output`)
      res.length.should.equal(1, `No L2 Tx rows!`)
      verifyL2TxOutput(res[0], tx)
    })
  })

  describe('tryBuildCanonicalChainBatchNotPresentOnL1', () => {
    it('Should not build a batch without tx outputs', async () => {
      const batchNum = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
        1,
        10
      )
      batchNum.should.equal(-1, `No batch should have been built`)

      const res = await postgres.select(`SELECT * FROM canonical_chain_batch`)
      res.length.should.equal(0, `No batch should exist`)
    })

    it('Should not build a batch without fewer than min tx outputs', async () => {
      const tx = createTxOutput(keccak256FromUtf8('tx'))
      await dataService.insertL2TransactionOutput(tx)

      const batchNum = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
        2,
        10
      )
      batchNum.should.equal(-1, `No batch should have been built`)

      const res = await postgres.select(`SELECT * FROM canonical_chain_batch`)
      res.length.should.equal(0, `No batch should exist`)
    })

    it('Should build a batch with min tx outputs', async () => {
      const tx = createTxOutput(keccak256FromUtf8('tx'))
      await dataService.insertL2TransactionOutput(tx)

      const batchNum = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const res = await postgres.select(`SELECT * FROM canonical_chain_batch`)
      res.length.should.equal(1, `Batch should exist`)
      res[0]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Wrong batch status!`
      )

      const txRes = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE canonical_chain_batch_number = ${batchNum}`
      )
      txRes.length.should.equal(1, `Should have batched 1 transaction`)
    })

    it('Should build a batch with 2 tx outputs with same timestamp', async () => {
      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        keccak256FromUtf8(defaultStateRoot),
        blockNumber + 1
      )
      await dataService.insertL2TransactionOutput(tx1)
      await dataService.insertL2TransactionOutput(tx2)

      const batchNum = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const batchRes = await postgres.select(
        `SELECT * FROM canonical_chain_batch`
      )
      batchRes.length.should.equal(1, `Batch should exist`)
      batchRes[0]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Wrong batch status!`
      )

      const txRes = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE canonical_chain_batch_number = ${batchNum}`
      )
      txRes.length.should.equal(2, `Should have batched 2 transactions`)
    })

    it('Should build 2 batches, given 2 tx outputs with different timestamps', async () => {
      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber,
        1
      )
      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        keccak256FromUtf8(defaultStateRoot),
        blockNumber + 1,
        2
      )
      await dataService.insertL2TransactionOutput(tx1)
      await dataService.insertL2TransactionOutput(tx2)

      const batchNum = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const batchRes = await postgres.select(
        `SELECT * FROM canonical_chain_batch`
      )
      batchRes.length.should.equal(1, `Batch should exist`)
      batchRes[0]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Wrong batch status!`
      )

      const txRes = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE canonical_chain_batch_number = ${batchNum}`
      )
      txRes.length.should.equal(1, `Should have batched 1 transaction`)

      const secondBatchNum = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
        1,
        10
      )
      secondBatchNum.should.equal(1, `Batch should have been built`)

      const secondBatchRes = await postgres.select(
        `SELECT * FROM canonical_chain_batch`
      )
      secondBatchRes.length.should.equal(2, `Batch should exist`)
      secondBatchRes[0]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Wrong batch status!`
      )
      secondBatchRes[1]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Wrong batch status [batch 2]!`
      )

      const txRes2 = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE canonical_chain_batch_number = ${secondBatchNum}`
      )
      txRes2.length.should.equal(
        1,
        `Should have batched 1 transaction in batch 2`
      )
    })

    it('Should cut off a batch at the max size', async () => {
      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        keccak256FromUtf8(defaultStateRoot),
        blockNumber + 1
      )
      await dataService.insertL2TransactionOutput(tx1)
      await dataService.insertL2TransactionOutput(tx2)

      const batchNum = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
        1,
        1
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const batchRes = await postgres.select(
        `SELECT * FROM canonical_chain_batch`
      )
      batchRes.length.should.equal(1, `Batch should exist`)
      batchRes[0]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Wrong batch status!`
      )

      const txRes = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE canonical_chain_batch_number = ${batchNum}`
      )
      txRes.length.should.equal(1, `Should have batched 1 transaction`)

      const secondBatchNum = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
        1,
        10
      )
      secondBatchNum.should.equal(1, `Batch should have been built`)

      const secondBatchRes = await postgres.select(
        `SELECT * FROM canonical_chain_batch`
      )
      secondBatchRes.length.should.equal(2, `Batch should exist`)
      secondBatchRes[0]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Wrong batch status!`
      )
      secondBatchRes[1]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Wrong batch status [batch 2]!`
      )

      const txRes2 = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE canonical_chain_batch_number = ${secondBatchNum}`
      )
      txRes2.length.should.equal(
        1,
        `Should have batched 1 transaction in batch 2`
      )
    })
  })
})
