/* External Imports */
import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import { RollupTransaction, TransactionAndRoot } from '../types'

export interface L1DataService {
  /**
   * Inserts the provided block into the associated RDB.
   *
   * @param block The Block to insert.
   * @param processed Whether or not the Block is completely processed and ready for use by other parts of the system.
   * @throws An error if there is a DB error.
   */
  insertBlock(block: Block, processed: boolean): Promise<void>

  /**
   * Atomically inserts the provided transactions into the associated RDB.
   *
   * @param transactions The transactions to insert.
   * @throws An error if there is a DB error.
   */
  insertTransactions(transactions: TransactionResponse[]): Promise<void>

  /**
   * Atomically inserts the provided block & contained transactions of interest.
   *
   * @param block The block to insert
   * @param txs The transactions to insert (may not be all of the txs in the associated block)
   * @param processed Whether or not the Block is completely processed and ready for use by other parts of the system.
   * @throws An error if there is a DB error.
   */
  insertBlockAndTransactions(
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
   * @returns The inserted transaction batch number.
   * @throws An error if there is a DB error.
   */
  insertRollupTransactions(
    l1TxHash: string,
    rollupTransactions: RollupTransaction[]
  ): Promise<number>

  /**
   * Atomically inserts the provided State Roots, creating a batch for them.
   *
   * @param l1TxHash The L1 Transaction hash.
   * @param stateRoots The state roots to insert.
   * @returns The inserted state root batch number.
   * @throws An error if there is a DB error.
   */
  insertRollupStateRoots(
    l1TxHash: string,
    stateRoots: string[]
  ): Promise<number>
}
