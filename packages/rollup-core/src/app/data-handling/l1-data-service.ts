/* External Imports */
import { RDB } from '@eth-optimism/core-db'

import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import { L1DataService, RollupTransaction } from '../../types'
import {
  blockInsertStatement,
  getBlockInsertValue,
  getRollupTransactionInsertValue,
  getTransactionInsertValue,
  rollupTxInsertStatement,
  txInsertStatement,
} from './query-utils'

export class DefaultL1DataService implements L1DataService {
  constructor(private readonly rdb: RDB) {}

  // TODO: All inserts below assume data is trusted and not malicious -- there is no SQL Injection protection.
  //  If this is not a safe assumption, we have the bigger problem of not being able to trust our block data.

  /**
   * @inheritDoc
   */
  public async insertBlock(
    block: Block,
    processed: boolean = false
  ): Promise<void> {
    return this.rdb.execute(
      `${blockInsertStatement} VALUES (${getBlockInsertValue(
        block,
        processed
      )})`
    )
  }

  /**
   * @inheritDoc
   */
  public async insertTransactions(
    transactions: TransactionResponse[]
  ): Promise<void> {
    if (!transactions || !transactions.length) {
      return
    }
    const values: string[] = transactions.map(
      (x) => `(${getTransactionInsertValue(x)})`
    )
    return this.rdb.execute(`${txInsertStatement} VALUES ${values.join(',')}`)
  }

  /**
   * @inheritDoc
   */
  public async insertBlockAndTransactions(
    block: Block,
    txs: TransactionResponse[],
    processed: boolean = false
  ): Promise<void> {
    await this.rdb.begin()
    try {
      await this.insertBlock(block, processed)
      await this.insertTransactions(txs)
    } catch (e) {
      await this.rdb.rollback()
      throw e
    }
    return this.rdb.commit()
  }

  /**
   * @inheritDoc
   */
  public async insertRollupTransactions(
    rollupTransactions: RollupTransaction[]
  ): Promise<void> {
    if (!rollupTransactions || !rollupTransactions.length) {
      return
    }
    const values: string[] = rollupTransactions.map(
      (x) => `(${getRollupTransactionInsertValue(x)})`
    )
    return this.rdb.execute(
      `${rollupTxInsertStatement} VALUES ${values.join(',')}`
    )
  }

  /**
   * @inheritDoc
   */
  public async updateBlockToProcessed(block_hash: string): Promise<void> {
    return this.rdb.execute(`
    UPDATE block 
    SET processed = TRUE 
    WHERE block_hash = ${block_hash}`)
  }
}
