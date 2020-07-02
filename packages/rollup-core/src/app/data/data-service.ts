/* External Imports */
import { RDB, Row } from '@eth-optimism/core-db'
import { getLogger, logError } from '@eth-optimism/core-utils'

import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import {
  DataService,
  RollupTransaction,
  TransactionAndRoot,
  VerificationCandidate,
} from '../../types'
import {
  blockInsertStatement,
  getBlockInsertValue,
  getL2TransactionInsertValue,
  getRollupStateRootInsertValue,
  getRollupTransactionInsertValue,
  getTransactionInsertValue,
  l2TransactionInsertStatement,
  rollupStateRootInsertStatement,
  rollupTxInsertStatement,
  txInsertStatement,
} from './query-utils'

const log = getLogger('data-service')

export class DefaultDataService implements DataService {
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
    await this.rdb.startTransaction()
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
    l1TxHash: string,
    rollupTransactions: RollupTransaction[]
  ): Promise<number> {
    if (!rollupTransactions || !rollupTransactions.length) {
      return
    }

    let batchNumber
    await this.rdb.startTransaction()
    try {
      batchNumber = await this.insertNewL1TransactionBatch(
        rollupTransactions[0].l1TxHash
      )

      const values: string[] = rollupTransactions.map(
        (x) => `(${getRollupTransactionInsertValue(x, batchNumber)})`
      )
      await this.rdb.execute(
        `${rollupTxInsertStatement} VALUES ${values.join(',')}`
      )

      await this.rdb.commit()
      return batchNumber
    } catch (e) {
      logError(
        log,
        `Error inserting rollup tx batch #${batchNumber}, l1 Tx Hash: ${l1TxHash}, batch: ${JSON.stringify(
          rollupTransactions
        )}!`,
        e
      )
      await this.rdb.rollback()
    }
  }

  /**
   * @inheritDoc
   */
  public async insertRollupStateRoots(
    l1TxHash: string,
    stateRoots: string[]
  ): Promise<number> {
    if (!stateRoots || !stateRoots.length) {
      return
    }

    let batchNumber
    await this.rdb.startTransaction()
    try {
      batchNumber = await this.insertNewL1StateRootBatch(l1TxHash)

      const values: string[] = stateRoots.map(
        (root, i) => `(${getRollupStateRootInsertValue(root, batchNumber, i)})`
      )
      await this.rdb.execute(
        `${rollupStateRootInsertStatement} VALUES ${values.join(',')}`
      )

      await this.rdb.commit()
      return batchNumber
    } catch (e) {
      logError(
        log,
        `Error inserting rollup state root batch #${batchNumber}, l1TxHash: ${l1TxHash}!`,
        e
      )
      await this.rdb.rollback()
    }
  }

  /**
   * @inheritDoc
   */
  public async updateBlockToProcessed(blockHash: string): Promise<void> {
    return this.rdb.execute(`
    UPDATE l1_block 
    SET processed = TRUE 
    WHERE block_hash = ${blockHash}`)
  }

  /*******************
   * L2 DATA SERVICE *
   *******************/

  /**
   * @inheritDoc
   */
  public async insertL2Transaction(tx: TransactionAndRoot): Promise<void> {
    return this.rdb.execute(
      `${l2TransactionInsertStatement} VALUES (${getL2TransactionInsertValue(
        tx
      )})`
    )
  }

  /************
   * VERIFIER *
   ************/

  /**
   * @inheritDoc
   */
  public async getVerificationCandidate(): Promise<VerificationCandidate> {
    const rows: Row[] = await this.rdb.select(`
      SELECT l1.batch_number as l1_batch, l2.batch_number as l2_batch, l1.batch_index, l1.state_root as l1_root, l2.state_root as l2_root
      FROM next_l1_batch l1
        LEFT OUTER JOIN next_l2_batch l2 
        ON l1.batch_number = l2.batch_number AND l1.batch_index = l2.batch_index
      ORDER BY l1.batch_index ASC
    `)

    if (!rows || !rows.length) {
      return undefined
    }

    return {
      l1BatchNumber: rows[0].columns['l1_batch'],
      l2BatchNumber: rows[0].columns['l2_batch'],
      roots: rows.map((x) => {
        return {
          l1Root: x.columns['l1_root'],
          l2Root: x.columns['l2_root'],
        }
      }),
    }
  }

  /**
   * @inheritDoc
   */
  public async verifyBatch(batchNumber): Promise<void> {
    await this.rdb.startTransaction()

    try {
      await this.rdb.commit()
    } catch (e) {
      await this.rdb.rollback()
    }
  }

  /***********
   * HELPERS *
   ***********/

  /**
   * @inheritDoc
   */
  protected async insertNewL1TransactionBatch(
    l1TxHash: string
  ): Promise<number> {
    let batchNumber: number

    let retries = 3
    // This should never fail, but adding in retries anyway
    while (retries > 0) {
      try {
        batchNumber = (await this.getMaxL1TxBatchNumber()) + 1
        await this.rdb.execute(`
            INSERT INTO l1_tx_batch(l1_tx_hash, batch_number) 
            VALUES ('${l1TxHash}', ${batchNumber})`)
        break
      } catch (e) {
        retries--
      }
    }

    return batchNumber
  }

  /**
   * @inheritDoc
   */
  protected async insertNewL1StateRootBatch(l1TxHash: string): Promise<number> {
    let batchNumber: number

    let retries = 3
    // This should never fail, but adding in retries anyway
    while (retries > 0) {
      try {
        batchNumber = (await this.getMaxL1StateRootBatchNumber()) + 1
        await this.rdb.execute(`
            INSERT INTO l1_state_root_batch(l1_tx_hash, batch_number) 
            VALUES ('${l1TxHash}', ${batchNumber})`)
        break
      } catch (e) {
        retries--
      }
    }

    return batchNumber
  }

  /**
   * Fetches the max L1 tx batch number for use in inserting a new tx batch
   * @returns The max batch number at the time of this query.
   */
  protected async getMaxL1TxBatchNumber(): Promise<number> {
    const rows = await this.rdb.select(
      `SELECT MAX(batch_number) as batch_number 
        FROM l1_tx_batch`
    )
    if (
      rows &&
      !!rows.length &&
      !!rows[0].columns &&
      !!rows[0].columns['batch_number']
    ) {
      // TODO: make sure we don't need to cast
      return rows[0].columns['batch_number']
    }

    return -1
  }

  /**
   * Fetches the max L1 state root batch number for use in inserting a new state root batch.
   * @returns The max batch number at the time of this query.
   */
  protected async getMaxL1StateRootBatchNumber(): Promise<number> {
    const rows = await this.rdb.select(
      `SELECT MAX(batch_number) as batch_number 
        FROM l1_state_root_batch`
    )
    if (
      rows &&
      !!rows.length &&
      !!rows[0].columns &&
      !!rows[0].columns['batch_number']
    ) {
      // TODO: make sure we don't need to cast
      return rows[0].columns['batch_number']
    }

    return -1
  }
}
