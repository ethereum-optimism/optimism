import '../setup'

/* External Imports */
import { PostgresDB, Row } from '@eth-optimism/core-db'
import { keccak256FromUtf8 } from '@eth-optimism/core-utils'

/* Internal Imports */
import { DefaultDataService } from '../../src/app/data'
import {
  blockNumber,
  createRollupTx,
  createTx,
  createTxOutput,
  defaultStateRoot,
  l1Block,
  verifyL1BlockRes,
  verifyL2TxOutput,
} from './helpers'
import { BatchSubmissionStatus, QueueOrigin } from '../../src/types/data'

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

      const status = await postgres.select(
        `SELECT status FROM canonical_chain_batch WHERE batch_number = ${batchNum}`
      )
      status.length.should.equal(1, `Only one batch should be created`)
      status[0]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Batch should be queued!`
      )
    })

    it('Should not build a batch with min tx outputs from L1', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [l1Tx])
      const rTx1 = createRollupTx(l1Tx, QueueOrigin.SAFETY_QUEUE)
      const rTx2 = createRollupTx(l1Tx, QueueOrigin.SAFETY_QUEUE, 0, 1)
      const submissionIndex = await dataService.insertL1RollupTransactions(
        l1Tx.hash,
        [rTx1, rTx2],
        true
      )

      const rollupTxRes = await postgres.select(
        `SELECT id FROM l1_rollup_tx LIMIT 1`
      )

      const tx = createTxOutput(keccak256FromUtf8('tx'))
      tx.l1RollupTransactionId = parseInt(rollupTxRes[0]['id'], 10)
      await dataService.insertL2TransactionOutput(tx)

      const batchNum = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
        1,
        10
      )
      batchNum.should.equal(-1, `Batch should not have been built`)

      const res = await postgres.select(`SELECT * FROM canonical_chain_batch`)
      res.length.should.equal(0, `Batch should not exist`)
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

  describe('isNextStateCommitmentChainBatchToBuildAlreadyAppendedOnL1', () => {
    it('Should return false when l1 and l2 have same number', async () => {
      const res = await dataService.isNextStateCommitmentChainBatchToBuildAlreadyAppendedOnL1()
      res.should.equal(false, `No state commitments should be on L1`)
    })

    it('Should return false when l2 is ahead of L1', async () => {
      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx1)

      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batchNum.should.equal(0, `first batch should have been built`)

      const res = await dataService.isNextStateCommitmentChainBatchToBuildAlreadyAppendedOnL1()
      res.should.equal(false, `L2 should be ahead of L1!`)
    })

    it('Should return true when l1 is ahead of L2', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [l1Tx], true)

      const batchNum = await dataService.insertL1RollupStateRoots(l1Tx.hash, [
        keccak256FromUtf8('hash 1'),
        keccak256FromUtf8('hash 2'),
      ])
      batchNum.should.equal(0, `First batch should be created!`)

      const res = await dataService.isNextStateCommitmentChainBatchToBuildAlreadyAppendedOnL1()
      res.should.equal(true, `L2 should be ahead of L1!`)
    })
  })

  describe('tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch', () => {
    it('Should not build a state commitment batch when 0 present in either', async () => {
      const batchNum = await dataService.tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch()
      batchNum.should.equal(-1, `No batch should have been built`)
    })

    it('Should not build a state commitment batch when more roots in L1 batch than present in L2 Tx Outputs', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [l1Tx], true)

      const l1StateRootBatchNum = await dataService.insertL1RollupStateRoots(
        l1Tx.hash,
        [keccak256FromUtf8('hash 1'), keccak256FromUtf8('hash 2')]
      )
      l1StateRootBatchNum.should.equal(
        0,
        `L1 State Root Batch should have been created!`
      )

      const batchNum = await dataService.tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch()
      batchNum.should.equal(-1, `No batch should have been built`)
    })

    it('Should build a state commitment batch when exactly the same in L1 state root batch as L2 Tx Outputs', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [l1Tx], true)

      const l1StateRootBatchNum = await dataService.insertL1RollupStateRoots(
        l1Tx.hash,
        [keccak256FromUtf8('hash 1')]
      )
      l1StateRootBatchNum.should.equal(
        0,
        `L1 State Root Batch should have been created!`
      )

      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx1)

      const batchNum = await dataService.tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch()
      batchNum.should.equal(0, `Batch should have been built`)

      const status = await postgres.select(
        `SELECT status FROM state_commitment_chain_batch WHERE batch_number = ${batchNum}`
      )
      status.length.should.equal(1, `Only one batch should be created`)
      status[0]['status'].should.equal(
        BatchSubmissionStatus.FINALIZED,
        `Batch should be in FINALIZED status.`
      )
    })

    it('Should build batch 0 if 1 and 2 are on L1 but 0 on L2', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      const l1Tx2 = createTx(keccak256FromUtf8('tx 2'))
      await dataService.insertL1BlockAndTransactions(
        l1Block,
        [l1Tx, l1Tx2],
        true
      )

      let l1StateRootBatchNum = await dataService.insertL1RollupStateRoots(
        l1Tx.hash,
        [keccak256FromUtf8('hash 1')]
      )
      l1StateRootBatchNum.should.equal(
        0,
        `L1 State Root Batch should have been created!`
      )

      l1StateRootBatchNum = await dataService.insertL1RollupStateRoots(
        l1Tx2.hash,
        [keccak256FromUtf8('hash 2')]
      )
      l1StateRootBatchNum.should.equal(
        1,
        `L1 State Root Batch 1 should have been created!`
      )

      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx1)

      const batchNum = await dataService.tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch()
      batchNum.should.equal(0, `Batch should have been built`)
    })

    it('Should only include L1 batch size, even if L2 has more txs', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [l1Tx], true)

      const l1StateRootBatchNum = await dataService.insertL1RollupStateRoots(
        l1Tx.hash,
        [keccak256FromUtf8('hash 1')]
      )
      l1StateRootBatchNum.should.equal(
        0,
        `L1 State Root Batch should have been created!`
      )

      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx1)

      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx2)

      const batchNum = await dataService.tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch()
      batchNum.should.equal(0, `Batch should have been built`)

      const l2TxsBatched = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE state_commitment_chain_batch_number = ${l1StateRootBatchNum}`
      )
      l2TxsBatched.length.should.equal(
        1,
        `Only one tx should have been batched!`
      )
      l2TxsBatched[0]['tx_hash'].should.equal(
        tx1.transactionHash,
        `First tx should be the batched one!`
      )
    })
  })

  describe('tryBuildL2OnlyStateCommitmentChainBatch', () => {
    it('Should not build a state commitment batch when 0 txs present', async () => {
      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batchNum.should.equal(-1, `No batch should have been built`)
    })

    it('Should not build a state commitment batch when less than min num txs present', async () => {
      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx1)

      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        2,
        10
      )
      batchNum.should.equal(-1, `No batch should have been built`)
    })

    it('Should build a state commitment batch when min num txs present', async () => {
      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx1)

      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const status = await postgres.select(
        `SELECT status FROM state_commitment_chain_batch WHERE batch_number = ${batchNum}`
      )
      status.length.should.equal(1, `Only one batch should be created`)
      status[0]['status'].should.equal(
        BatchSubmissionStatus.QUEUED,
        `Batch should be in QUEUED status.`
      )
    })

    it('Should build a state commitment batch when more than min num txs present', async () => {
      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx1)

      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx2)

      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const count = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE state_commitment_chain_batch_number = ${batchNum}`
      )
      count.length.should.equal(2, `Both txos should have been batched!`)
    })

    it('Should build a state commitment batch with no more than the max num txs', async () => {
      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx1)

      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        defaultStateRoot,
        blockNumber
      )
      await dataService.insertL2TransactionOutput(tx2)

      let batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        1
      )
      batchNum.should.equal(0, `Batch should have been built`)

      let count = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE state_commitment_chain_batch_number = ${batchNum}`
      )
      count.length.should.equal(1, `First txo should have been batched!`)
      count[0]['tx_hash'].should.equal(
        tx1.transactionHash,
        `first batch should be tx 1!`
      )

      batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(1, 1)
      batchNum.should.equal(1, `Batch should have been built`)

      count = await postgres.select(
        `SELECT * FROM l2_tx_output WHERE state_commitment_chain_batch_number = ${batchNum}`
      )
      count.length.should.equal(1, `Second txo should have been batched!`)
      count[0]['tx_hash'].should.equal(
        tx2.transactionHash,
        `second batch should be tx 2!`
      )
    })
  })
})
