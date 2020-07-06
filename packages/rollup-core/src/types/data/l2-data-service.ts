/* Internal Imports */
import { TransactionAndRoot } from '../types'

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
}
