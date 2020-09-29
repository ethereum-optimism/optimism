import '../setup'

/* External Imports */
import { PostgresDB } from '@eth-optimism/core-db'
import { keccak256FromUtf8 } from '@eth-optimism/core-utils'

/* Internal Imports */
import { DefaultDataService } from '../../src/app/data'
import {
  blockNumber,
  createTx,
  createTxOutput,
  defaultStateRoot,
  deleteAllData,
  insertTxOutput,
  l1Block,
} from './helpers'
import { BatchSubmissionStatus, VerificationStatus } from '../../src/types/data'

describe('Verifier Data Data Service (will fail if postgres is not running with expected schema)', () => {
  let dataService: DefaultDataService
  let postgres: PostgresDB
  before(async () => {
    postgres = new PostgresDB('0.0.0.0', 5432, 'test', 'test', 'rollup')
    dataService = new DefaultDataService(postgres)
  })

  beforeEach(async () => {
    await deleteAllData(postgres)
  })

  describe('getNextVerificationCandidate', () => {
    it('Should not return candidate if no data exists', async () => {
      const candidate = await dataService.getNextVerificationCandidate()
      const candidateExists = !!candidate
      candidateExists.should.equal(
        false,
        `No verification candidate should exist!`
      )
    })

    it('Should not return candidate if data only exists in L1', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [tx])
      const stateRoots = [
        keccak256FromUtf8('root1'),
        keccak256FromUtf8('root2'),
        keccak256FromUtf8('root3'),
      ]
      const batchNumber = await dataService.insertL1RollupStateRoots(
        tx.hash,
        stateRoots
      )

      const candidate = await dataService.getNextVerificationCandidate()
      const candidateExists = !!candidate
      candidateExists.should.equal(
        false,
        `No verification candidate should exist!`
      )
    })

    it('Should not return candidate if data only exists in L2', async () => {
      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await insertTxOutput(
        dataService,
        tx1,
        BatchSubmissionStatus.FINALIZED,
        BatchSubmissionStatus.QUEUED
      )

      const candidate = await dataService.getNextVerificationCandidate()
      const candidateExists = !!candidate
      candidateExists.should.equal(
        false,
        `No verification candidate should exist!`
      )
    })

    it('Should return candidate if data exists in both L1 and L2', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [l1Tx], true)

      const l1BatchNum = await dataService.insertL1RollupStateRoots(l1Tx.hash, [
        keccak256FromUtf8('hash 1'),
        keccak256FromUtf8('hash 2'),
        keccak256FromUtf8('hash 3'),
      ])
      l1BatchNum.should.equal(0, `L1 batch should have been created!`)

      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await insertTxOutput(dataService, tx1, BatchSubmissionStatus.FINALIZED)
      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        keccak256FromUtf8(defaultStateRoot),
        blockNumber + 1
      )
      await insertTxOutput(dataService, tx2, BatchSubmissionStatus.FINALIZED)

      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const candidate = await dataService.getNextVerificationCandidate()
      const candidateExists = !!candidate
      candidateExists.should.equal(true, `Verification candidate should exist!`)
      candidate.roots.length.should.equal(
        2,
        `There should be 2 roots in the verification candidate!`
      )
      candidate.batchNumber.should.equal(
        '0',
        `The batch number for verification candidate should be 0!`
      )
    })

    it('Should not return candidate if data exists in both L1 and L2 but verified in L1', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [l1Tx], true)

      const l1BatchNum = await dataService.insertL1RollupStateRoots(l1Tx.hash, [
        keccak256FromUtf8('hash 1'),
        keccak256FromUtf8('hash 2'),
        keccak256FromUtf8('hash 3'),
      ])
      l1BatchNum.should.equal(0, `L1 batch should have been created!`)

      await dataService.verifyStateRootBatch(0)

      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await insertTxOutput(dataService, tx1, BatchSubmissionStatus.FINALIZED)
      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        keccak256FromUtf8(defaultStateRoot),
        blockNumber + 1
      )
      await insertTxOutput(dataService, tx2, BatchSubmissionStatus.FINALIZED)

      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const candidate = await dataService.getNextVerificationCandidate()
      const candidateExists = !!candidate
      candidateExists.should.equal(
        false,
        `Verification candidate should not exist!`
      )
    })

    it('Should not return candidate if data exists in both L1 and L2 but Fraudulent in L1', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [l1Tx], true)

      const l1BatchNum = await dataService.insertL1RollupStateRoots(l1Tx.hash, [
        keccak256FromUtf8('hash 1'),
        keccak256FromUtf8('hash 2'),
        keccak256FromUtf8('hash 3'),
      ])
      l1BatchNum.should.equal(0, `L1 batch should have been created!`)

      await postgres.execute(
        `UPDATE l1_rollup_state_root_batch SET status = '${VerificationStatus.FRAUDULENT}' WHERE batch_number = ${l1BatchNum}`
      )

      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await insertTxOutput(dataService, tx1, BatchSubmissionStatus.FINALIZED)
      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        keccak256FromUtf8(defaultStateRoot),
        blockNumber + 1
      )
      await insertTxOutput(dataService, tx2, BatchSubmissionStatus.FINALIZED)

      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const candidate = await dataService.getNextVerificationCandidate()
      const candidateExists = !!candidate
      candidateExists.should.equal(
        false,
        `Verification candidate should not exist!`
      )
    })

    it('Should not return candidate if data exists in both L1 and L2 but Removed in L1', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [l1Tx], true)

      const l1BatchNum = await dataService.insertL1RollupStateRoots(l1Tx.hash, [
        keccak256FromUtf8('hash 1'),
        keccak256FromUtf8('hash 2'),
        keccak256FromUtf8('hash 3'),
      ])
      l1BatchNum.should.equal(0, `L1 batch should have been created!`)

      await postgres.execute(
        `UPDATE l1_rollup_state_root_batch SET status = '${VerificationStatus.REMOVED}' WHERE batch_number = ${l1BatchNum}`
      )

      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await insertTxOutput(dataService, tx1, BatchSubmissionStatus.FINALIZED)
      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        keccak256FromUtf8(defaultStateRoot),
        blockNumber + 1
      )
      await insertTxOutput(dataService, tx2, BatchSubmissionStatus.FINALIZED)

      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const candidate = await dataService.getNextVerificationCandidate()
      const candidateExists = !!candidate
      candidateExists.should.equal(
        false,
        `Verification candidate should not exist!`
      )
    })

    it('Should return next candidate after previous candidate is verified', async () => {
      const l1Tx = createTx(keccak256FromUtf8('tx 1'))
      const l1Tx2 = createTx(keccak256FromUtf8('tx 2'))
      await dataService.insertL1BlockAndTransactions(
        l1Block,
        [l1Tx, l1Tx2],
        true
      )

      const l1BatchNum = await dataService.insertL1RollupStateRoots(l1Tx.hash, [
        keccak256FromUtf8('hash 1'),
        keccak256FromUtf8('hash 2'),
        keccak256FromUtf8('hash 3'),
      ])
      l1BatchNum.should.equal(0, `L1 batch should have been created!`)

      await dataService.verifyStateRootBatch(0)

      const tx1 = createTxOutput(
        keccak256FromUtf8('tx 1'),
        defaultStateRoot,
        blockNumber
      )
      await insertTxOutput(dataService, tx1, BatchSubmissionStatus.FINALIZED)
      const tx2 = createTxOutput(
        keccak256FromUtf8('tx 2'),
        keccak256FromUtf8(defaultStateRoot),
        blockNumber + 1
      )
      await insertTxOutput(dataService, tx2, BatchSubmissionStatus.FINALIZED)
      const tx3 = createTxOutput(
        keccak256FromUtf8('tx 3'),
        keccak256FromUtf8(defaultStateRoot),
        blockNumber + 2
      )
      await insertTxOutput(dataService, tx3, BatchSubmissionStatus.FINALIZED)

      const batchNum = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batchNum.should.equal(0, `Batch should have been built`)

      const stateRoot4: string = keccak256FromUtf8('hash 4')
      const stateRoot5: string = keccak256FromUtf8('hash 5')

      const tx4 = createTxOutput(
        keccak256FromUtf8('tx 4'),
        stateRoot4,
        blockNumber + 3
      )
      await insertTxOutput(dataService, tx4, BatchSubmissionStatus.FINALIZED)

      const tx5 = createTxOutput(
        keccak256FromUtf8('tx 5'),
        stateRoot5,
        blockNumber + 4
      )
      await insertTxOutput(dataService, tx5, BatchSubmissionStatus.FINALIZED)

      const batch2Num = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
        1,
        10
      )
      batch2Num.should.equal(1, `Batch should have been built`)

      const l1Batch2 = await dataService.insertL1RollupStateRoots(l1Tx2.hash, [
        stateRoot4,
        stateRoot5,
      ])
      l1Batch2.should.equal(1, `L1 batch should have been created!`)

      const candidate = await dataService.getNextVerificationCandidate()
      const candidateExists = !!candidate
      candidateExists.should.equal(true, `Verification candidate should exist!`)
      candidate.batchNumber.should.equal('1', `Wrong batch number!`)
      candidate.roots.length.should.equal(2, `Wrong batch size`)
      candidate.roots[0].gethRoot.should.equal(
        stateRoot4,
        `Geth state root 4 wrong!`
      )
      candidate.roots[0].l1Root.should.equal(
        stateRoot4,
        `L1 State root 4 wrong!`
      )
      candidate.roots[1].gethRoot.should.equal(
        stateRoot5,
        `Geth state root 5 wrong!`
      )
      candidate.roots[1].l1Root.should.equal(
        stateRoot5,
        `L1 State root 5 wrong!`
      )
    })
  })

  describe('verifyStateRootBatch', () => {
    it('Should not verify batch if given wrong batch number', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [tx])
      const stateRoots = [
        keccak256FromUtf8('root1'),
        keccak256FromUtf8('root2'),
        keccak256FromUtf8('root3'),
      ]
      const batchNumber = await dataService.insertL1RollupStateRoots(
        tx.hash,
        stateRoots
      )

      await dataService.verifyStateRootBatch(batchNumber + 1)

      const res = await postgres.select(
        `SELECT * FROM l1_rollup_state_root_batch WHERE batch_number = ${batchNumber}`
      )
      res.length.should.equal(1, `Incorrect result size!`)
      res[0]['status'].should.equal(
        VerificationStatus.UNVERIFIED,
        `Incorrect status!`
      )
    })

    it('Should not verify batch if batch is fraudulent', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [tx])
      const stateRoots = [
        keccak256FromUtf8('root1'),
        keccak256FromUtf8('root2'),
        keccak256FromUtf8('root3'),
      ]
      const batchNumber = await dataService.insertL1RollupStateRoots(
        tx.hash,
        stateRoots
      )

      await postgres.execute(
        `UPDATE l1_rollup_state_root_batch SET status = '${VerificationStatus.FRAUDULENT}' WHERE batch_number = ${batchNumber}`
      )

      await dataService.verifyStateRootBatch(batchNumber)

      const res = await postgres.select(
        `SELECT * FROM l1_rollup_state_root_batch WHERE batch_number = ${batchNumber}`
      )
      res.length.should.equal(1, `Incorrect result size!`)
      res[0]['status'].should.equal(
        VerificationStatus.FRAUDULENT,
        `Incorrect status!`
      )
    })

    it('Should not verify batch if batch is removed', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [tx])
      const stateRoots = [
        keccak256FromUtf8('root1'),
        keccak256FromUtf8('root2'),
        keccak256FromUtf8('root3'),
      ]
      const batchNumber = await dataService.insertL1RollupStateRoots(
        tx.hash,
        stateRoots
      )

      await postgres.execute(
        `UPDATE l1_rollup_state_root_batch SET status = '${VerificationStatus.REMOVED}' WHERE batch_number = ${batchNumber}`
      )

      await dataService.verifyStateRootBatch(batchNumber)

      const res = await postgres.select(
        `SELECT * FROM l1_rollup_state_root_batch WHERE batch_number = ${batchNumber}`
      )
      res.length.should.equal(1, `Incorrect result size!`)
      res[0]['status'].should.equal(
        VerificationStatus.REMOVED,
        `Incorrect status!`
      )
    })

    it('Should verify batch if batch is unverified and batch number is correct', async () => {
      const tx = createTx(keccak256FromUtf8('tx 1'))
      await dataService.insertL1BlockAndTransactions(l1Block, [tx])
      const stateRoots = [
        keccak256FromUtf8('root1'),
        keccak256FromUtf8('root2'),
        keccak256FromUtf8('root3'),
      ]
      const batchNumber = await dataService.insertL1RollupStateRoots(
        tx.hash,
        stateRoots
      )

      await dataService.verifyStateRootBatch(batchNumber)

      const res = await postgres.select(
        `SELECT * FROM l1_rollup_state_root_batch WHERE batch_number = ${batchNumber}`
      )
      res.length.should.equal(1, `Incorrect result size!`)
      res[0]['status'].should.equal(
        VerificationStatus.VERIFIED,
        `Incorrect status!`
      )
    })
  })
})
