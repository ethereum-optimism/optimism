/* External Imports */
import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import { RollupTransaction } from './types'

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
   * @param block_hash The block hash identifying the block to update.
   * @throws An error if there is a DB error.
   */
  updateBlockToProcessed(block_hash: string): Promise<void>

  /**
   * Atomically inserts the provided RollupTransactions.
   *
   * @param rollupTransactions The RollupTransactions to insert.
   * @throws An error if there is a DB error.
   */
  insertRollupTransactions(
    rollupTransactions: RollupTransaction[]
  ): Promise<void>
}
