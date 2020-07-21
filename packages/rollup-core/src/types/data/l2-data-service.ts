/* Internal Imports */
import { TransactionOutput } from '../types'
import {TransactionBatchSubmission, BatchSubmissionStatus, StateCommitmentBatchSubmission} from './types'

export interface L2DataService {
  /**
   * Inserts the provided L2 Transaction Output into the associated RDB.
   *
   * @param transaction The transaction to insert.
   * @throws An error if there is a DB error.
   */
  insertL2TransactionOutput(transaction: TransactionOutput): Promise<void>

  /**
   * Builds a Canonical Chain Tx batch for L2 Tx Outputs that are not present on L1
   * if there are unbatched L2 Transaction Outputs with different timestamps.
   *
   * @returns The number of the cc Batch that was built, or -1 if one wasn't built.
   * @throws An error if there is a DB error.
   */
  tryBuildCanonicalChainBatchNotPresentOnL1(): Promise<number>

  /**
   * Determines whether or not the next State Commitment Chain batch represents a set of
   * state roots that were already appended to the L1 chain.
   *
   * @returns true if the next batch to build was already appended, false otherwise.
   * @throws An error if there is a DB error.
   */
  wasNextStateCommitmentChainBatchToBuildAppendedOnL1(): Promise<boolean>
  
  /**
   * Attempts to build a State Commitment Chain batch to match the batch present on L1.
   *
   * @returns The batch number of the created batch if one was created or -1 if one was not created.
   * @throws An error if there is a DB error.
   */
  tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch(): Promise<number>

  /**
   * Attempts to build a State Commitment Chain batch of state roots not yet appended to L1.
   *
   * @param minBatchSize The min number of state roots to include in a batch.
   * @param maxBatchSize The max number of state roots to include in a batch.
   * @returns The batch number of the created batch if one was created or -1 if one was not created.
   * @throws An error if there is a DB error.
   */
  tryBuildL2OnlyStateCommitmentChainBatch(minBatchSize: number, maxBatchSize: number): Promise<number>

  /**
   * Gets the next Canonical Chain Tx batch for submission to L1, if one exists.
   *
   * @returns The TransactionBatchSubmission object, or undefined
   * @throws An error if there is a DB error.
   */
  getNextCanonicalChainTransactionBatchToSubmit(): Promise<TransactionBatchSubmission>

  /**
   * Marks the Canonical Chain Tx batch with the provided batch number as submitted to the L1 chain.
   *
   * @param ccBatchNumber The batch number to mark as submitted.
   * @param l1SubmissionTxHash The tx hash of this rollup batch submission tx on L1.
   * @throws An error if there is a DB error.
   */
  markTransactionBatchSubmittedToL1(
    ccBatchNumber: number,
    l1SubmissionTxHash: string
  ): Promise<void>

  /**
   * Marks the Canonical Chain Tx batch with the provided batch number as confirmed on the L1 chain.
   *
   * @param ccBatchNumber The batch number to mark as confirmed.
   * @param l1SubmissionTxHash The tx hash of this rollup batch submission tx on L1.
   * @throws An error if there is a DB error.
   */
  markTransactionBatchConfirmedOnL1(
    ccBatchNumber: number,
    l1SubmissionTxHash: string
  ): Promise<void>

  /**
   * Gets the next State Commitment batch for submission to L1, if one exists.
   *
   * @returns The StateCommitmentBatchSubmission object, or undefined
   * @throws An error if there is a DB error.
   */
  getNextStateCommitmentBatchToSubmit(): Promise<StateCommitmentBatchSubmission>

  /**
   * Marks the Canonical Chain Tx batch with the provided batch number as submitted to the L1 chain.
   *
   * @param ccBatchNumber The batch number to mark as submitted.
   * @param l1SubmissionTxHash The tx hash of this batch submission tx on L1.
   * @throws An error if there is a DB error.
   */
  markStateRootBatchSubmittedToL1(
    ccBatchNumber: number,
    l1SubmissionTxHash: string
  ): Promise<void>

  /**
   * Marks the Canonical Chain Tx batch with the provided batch number as confirmed on the L1 chain.
   *
   * @param ccBatchNumber The batch number to mark as confirmed.
   * @param l1SubmissionTxHash The tx hash of this batch submission tx on L1.
   * @throws An error if there is a DB error.
   */
  markStateRootBatchFinalOnL1(
    ccBatchNumber: number,
    l1SubmissionTxHash: string
  ): Promise<void>
}
