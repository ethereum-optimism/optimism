/* External Imports */
import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import {
  GethSubmission,
  L1BlockPersistenceInfo,
  RollupTransaction,
} from '../types'
import { GethSubmissionRecord } from './types'

export interface L1DataService {
  /**
   * Gets Information regarding whether or not the block (and its related data) associated with
   * the provided L1 block number is present in the DB.
   *
   * @param blockNumber The block number in question.
   * @returns The L1BlockPersistenceInfo object containing booleans indicating what has been persisted.
   */
  getL1BlockPersistenceInfo(
    blockNumber: number
  ): Promise<L1BlockPersistenceInfo>

  /**
   * Inserts the provided block into the associated RDB.
   *
   * @param block The Block to insert.
   * @param processed Whether or not the Block is completely processed and ready for use by other parts of the system.
   * @throws An error if there is a DB error.
   */
  insertL1Block(block: Block, processed: boolean): Promise<void>

  /**
   * Atomically inserts the provided transactions into the associated RDB.
   *
   * @param transactions The transactions to insert.
   * @throws An error if there is a DB error.
   */
  insertL1Transactions(transactions: TransactionResponse[]): Promise<void>

  /**
   * Atomically inserts the provided block & contained transactions of interest.
   *
   * @param block The block to insert
   * @param txs The transactions to insert (may not be all of the txs in the associated block)
   * @param processed Whether or not the Block is completely processed and ready for use by other parts of the system.
   * @throws An error if there is a DB error.
   */
  insertL1BlockAndTransactions(
    block: Block,
    txs: TransactionResponse[],
    processed: boolean
  ): Promise<void>

  /**
   * Updates the block with the provided block_hash to be marked as "processed," signifying that all data
   * associated with it is present and ready for consumption.
   *
   * @param blockHash The block hash identifying the block to update.
   * @throws An error if there is a DB error.
   */
  updateBlockToProcessed(blockHash: string): Promise<void>

  /**
   * Atomically inserts the provided RollupTransactions, creating a batch for them.
   *
   * @param l1TxHash The L1 Transaction hash.
   * @param rollupTransactions The RollupTransactions to insert.
   * @param queueForGethSubmission Whether or not to queue the provided RollupTransactions for submission to Geth.
   * @returns The inserted geth submission queue index if queued for geth submission.
   * @throws An error if there is a DB error.
   */
  insertL1RollupTransactions(
    l1TxHash: string,
    rollupTransactions: RollupTransaction[],
    queueForGethSubmission?: boolean
  ): Promise<number>

  /**
   * Creates a Queued Geth Submission from the oldest un-queued transaction that is from the L1 To L2 queue.
   *
   * @param queueOrigins The next entry may only be from queue origins provided here (it's a filter).
   * @returns The created entry's queue index or undefined if no fitting L1ToL2 transaction exists.
   * @throws Error if there is a DB error
   */
  queueNextGethSubmission(queueOrigins: number[]): Promise<number>

  /**
   * Atomically inserts the provided State Roots, creating a rollup state root batch for them.
   *
   * @param l1TxHash The hash of the L1 Transaction that posted these state roots.
   * @param stateRoots The state roots to insert.
   * @returns The inserted state root batch number.
   * @throws An error if there is a DB error.
   */
  insertL1RollupStateRoots(
    l1TxHash: string,
    stateRoots: string[]
  ): Promise<number>

  /**
   * Fetches the next Queued Geth Submission from L1 to submit to L2, if there is one.
   *
   * @returns The fetched Queued Geth Submission or undefined if one is not present in the DB.
   */
  getNextQueuedGethSubmission(): Promise<GethSubmission>

  /**
   * Marks the provided Queued Geth Submission as submitted to L2.
   *
   * @params queueIndex The geth submission queue index to mark as submitted to L2.
   * @throws An error if there is a DB error.
   */
  markQueuedGethSubmissionSubmittedToGeth(queueIndex: number): Promise<void>
}
