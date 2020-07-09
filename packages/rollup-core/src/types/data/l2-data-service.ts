/* Internal Imports */
import { TransactionAndRoot } from '../types'
import { L1BatchSubmission, L2BatchStatus } from './types'

export interface L2DataService {
  /**
   * Inserts the provided L2 transaction into the associated RDB.
   *
   * @param transaction The transaction to insert.
   * @throws An error if there is a DB error.
   */
  insertL2Transaction(transaction: TransactionAndRoot): Promise<void>

  /**
   * Builds an L2-only batch if there are unbatched L2 Transactions with different timestamps.
   *
   * @returns The number of the L2 Batch that was built, or -1 if one wasn't built.
   */
  tryBuildL2OnlyBatch(): Promise<number>

  /**
   * Builds an L2 batch of the provided size matching the provided batch number
   * if there are enough L2 transactions to support it.
   * @param batchNumber The expected batch number
   * @param batchSize The expected batch size
   * @throws If there are multiple unbatched batches (based on timestamp) and the oldest is not
   * at least `batchNumber` in size (our L1 & L2 batches don't match).
   */
  tryBuildL2BatchToMatchL1(
    batchNumber: number,
    batchSize: number
  ): Promise<number>

  /**
   * Gets the next L2 Batch for submission to L1, if one exists.
   *
   * @returns The L1BatchSubmission object, or undefined
   * @throws An error if there is a DB error.
   */
  getNextBatchForL1Submission(): Promise<L1BatchSubmission>

  /**
   * Marks the tx batch with the provided batch number as submitted to the L1 chain.
   *
   * @param batchNumber The batch number to mark as submitted.
   * @param l1TxHash The L1 transaction hash for the batch submission.
   * @throws An error if there is a DB error.
   */
  markTransactionBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void>

  /**
   * Marks the tx batch with the provided batch number as confirmed on the L1 chain.
   *
   * @param batchNumber The batch number to mark as confirmed.
   * @param l1TxHash The L1 transaction hash for the batch submission.
   * @throws An error if there is a DB error.
   */
  markTransactionBatchConfirmedOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void>

  /**
   * Marks the state root batch with the provided batch number as submitted to the L1 chain.
   *
   * @param batchNumber The batch number to mark as submitted.
   * @param l1TxHash The L1 transaction hash for the batch submission.
   * @throws An error if there is a DB error.
   */
  markStateRootBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void>

  /**
   * Marks the state root batch with the provided batch number as confirmed on the L1 chain.
   *
   * @param batchNumber The batch number to mark as confirmed.
   * @param l1TxHash The L1 transaction hash for the batch submission.
   * @throws An error if there is a DB error.
   */
  markStateRootBatchConfirmedOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void>
}
