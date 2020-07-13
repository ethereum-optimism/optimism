/* External Imports */
import { RDB, Row } from '@eth-optimism/core-db'
import { add0x, getLogger, logError } from '@eth-optimism/core-utils'

import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import {
  BlockBatches,
  DataService,
  L1BatchRecord,
  L1BatchSubmission,
  L2BatchStatus,
  QueueOrigin,
  RollupTransaction,
  TransactionAndRoot,
  VerificationCandidate,
} from '../../types'
import {
  l1BlockInsertStatement,
  getL1BlockInsertValue,
  getL2TransactionInsertValue,
  getL1RollupStateRootInsertValue,
  getL1RollupTransactionInsertValue,
  getL1TransactionInsertValue,
  l2TransactionInsertStatement,
  l1RollupStateRootInsertStatement,
  l1RollupTxInsertStatement,
  l1TxInsertStatement,
} from './query-utils'

const log = getLogger('data-service')

export class DefaultDataService implements DataService {
  constructor(private readonly rdb: RDB) {}

  // TODO: All inserts below assume data is trusted and not malicious -- there is no SQL Injection protection.
  //  If this is not a safe assumption, we have the bigger problem of not being able to trust our block data.

  /**
   * @inheritDoc
   */
  public async insertL1Block(
    block: Block,
    processed: boolean = false
  ): Promise<void> {
    return this.rdb.execute(
      `${l1BlockInsertStatement} VALUES (${getL1BlockInsertValue(
        block,
        processed
      )})`
    )
  }

  /**
   * @inheritDoc
   */
  public async insertL1Transactions(
    transactions: TransactionResponse[]
  ): Promise<void> {
    if (!transactions || !transactions.length) {
      return
    }
    const values: string[] = transactions.map(
      (x) => `(${getL1TransactionInsertValue(x)})`
    )
    return this.rdb.execute(`${l1TxInsertStatement} VALUES ${values.join(',')}`)
  }

  /**
   * @inheritDoc
   */
  public async insertL1BlockAndTransactions(
    block: Block,
    txs: TransactionResponse[],
    processed: boolean = false
  ): Promise<void> {
    await this.rdb.startTransaction()
    try {
      await this.insertL1Block(block, processed)
      await this.insertL1Transactions(txs)
    } catch (e) {
      await this.rdb.rollback()
      throw e
    }
    return this.rdb.commit()
  }

  /**
   * @inheritDoc
   */
  public async insertL1RollupTransactions(
    l1TxHash: string,
    rollupTransactions: RollupTransaction[],
    createBatch: boolean = false
  ): Promise<number> {
    if (!rollupTransactions || !rollupTransactions.length) {
      return
    }

    let batchNumber: number
    await this.rdb.startTransaction()
    try {
      if (createBatch) {
        batchNumber = await this.insertNewL1TransactionBatch(
          rollupTransactions[0].l1TxHash
        )
      }

      const values: string[] = rollupTransactions.map(
        (x) => `(${getL1RollupTransactionInsertValue(x, batchNumber)})`
      )

      await this.rdb.execute(
        `${l1RollupTxInsertStatement} VALUES ${values.join(',')}`
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
  public async createNextL1ToL2Batch(): Promise<number> {
    const txHashRes = await this.rdb.select(`
      SELECT l1_tx_hash, l1_tx_log_index
      FROM rollup_tx
      WHERE batch_number IS NULL AND queue_origin = ${QueueOrigin.L1_TO_L2_QUEUE} 
      ORDER BY l1_block_number ASC, l1_tx_index ASC, l1_tx_log_index ASC 
      LIMIT 1
    `)
    if (!txHashRes || !txHashRes.length || !txHashRes[0].columns['tx_hash']) {
      return undefined
    }

    const txHash = txHashRes[0].columns['l1_tx_hash']
    const txLogIndex = txHashRes[0].columns['l1_tx_log_index']

    await this.rdb.startTransaction()
    try {
      const batchNumber: number = await this.insertNewL1TransactionBatch(txHash)

      await this.rdb.execute(`
        UPDATE rollup_tx
        SET batch_number = ${batchNumber}, batch_index = 0
        WHERE l1_tx_hash = '${txHash}' AND l1_tx_log_index = ${txLogIndex} 
      `)

      return batchNumber
    } catch (e) {
      logError(
        log,
        `Error executing createNextL1ToL2Batch for tx hash ${txHash}... rolling back`,
        e
      )
      await this.rdb.rollback()
      throw e
    }
  }

  /**
   * @inheritDoc
   */
  public async createNextSafetyQueueBatch(): Promise<number> {
    const txHashRes = await this.rdb.select(`
      SELECT l1_tx_hash, l1_tx_log_index
      FROM rollup_tx
      WHERE batch_number IS NULL AND queue_origin = ${QueueOrigin.SAFETY_QUEUE} 
      ORDER BY l1_block_number ASC, l1_tx_index ASC, l1_tx_log_index ASC 
      LIMIT 1
    `)
    if (!txHashRes || !txHashRes.length || !txHashRes[0].columns['tx_hash']) {
      return undefined
    }

    const txHash = txHashRes[0].columns['l1_tx_hash']
    const txLogIndex = txHashRes[0].columns['l1_tx_log_index']

    await this.rdb.startTransaction()
    try {
      const batchNumber: number = await this.insertNewL1TransactionBatch(txHash)

      await this.rdb.execute(`
        UPDATE rollup_tx
        SET batch_number = ${batchNumber}, batch_index = 0
        WHERE l1_tx_hash = '${txHash}' AND l1_tx_log_index = ${txLogIndex} 
      `)

      return batchNumber
    } catch (e) {
      logError(
        log,
        `Error executing createNextSafetyQueueBatch for tx hash ${txHash}... rolling back`,
        e
      )
      await this.rdb.rollback()
      throw e
    }
  }

  /**
   * @inheritDoc
   */
  public async insertL1RollupStateRoots(
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
        (root, i) =>
          `(${getL1RollupStateRootInsertValue(root, batchNumber, i)})`
      )
      await this.rdb.execute(
        `${l1RollupStateRootInsertStatement} VALUES ${values.join(',')}`
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
  public async getOldestUnverifiedL1TransactionBatch(): Promise<L1BatchRecord> {
    const res: Row[] = await this.rdb.select(`
      SELECT COUNT(*) as batch_size, batch_number, block_timestamp
      FROM next_l1_verification_batch 
      GROUP BY batch_number, block_timestamp
      ORDER BY batch_number ASC
    `) // note batch_number should be the same, just ordering in case

    if (!res || !res.length || !res[0].columns['batch_size']) {
      return undefined
    }
    return {
      batchSize: res[0].columns['batch_size'],
      batchNumber: res[0].columns['batch_number'],
      blockTimestamp: res[0].columns['block_timestamp'],
    }
  }

  /**
   * @inheritDoc
   */
  public async getNextBatchForL2Submission(): Promise<BlockBatches> {
    const res: Row[] = await this.rdb.select(`
      SELECT batch_number, target, calldata, block_timestamp, block_number, l1_tx_hash, l1_tx_index, l1_tx_log_index, queue_origin, sender, l1_message_sender, gas_limit, nonce, signature
      FROM next_l2_submission_batch
    `)

    if (!res || !res.length) {
      return undefined
    }

    const batchNumber = res[0].columns['batch_number']
    const timestamp = res[0].columns['block_timestamp']
    const blockNumber = res[0].columns['block_number']

    return {
      batchNumber,
      timestamp,
      blockNumber,
      batches: [
        res.map((row: Row, batchIndex: number) => {
          const tx: RollupTransaction = {
            batchIndex,
            l1TxHash: row.columns['l1_tx_hash'],
            l1TxIndex: row.columns['l1_tx_index'],
            l1TxLogIndex: row.columns['l1_tx_log_index'],
            target: row.columns['target'],
            calldata: row.columns['calldata'], // TODO: may have to format Buffer => string
            l1Timestamp: row.columns['block_timestamp'],
            l1BlockNumber: row.columns['block_number'],
            queueOrigin: row.columns['queue_origin'],
          }

          if (!!row.columns['sender']) {
            tx.sender = row.columns['sender']
          }
          if (!!row.columns['l1MessageSender']) {
            tx.l1MessageSender = row.columns['l1_message_sender']
          }
          if (!!row.columns['gas_limit']) {
            tx.gasLimit = row.columns['gas_limit']
          }
          if (!!row.columns['nonce']) {
            tx.nonce = row.columns['nonce']
          }
          if (!!row.columns['signature']) {
            tx.nonce = row.columns['signature']
          }
          return tx
        }),
      ],
    }
  }

  /**
   * @inheritDoc
   */
  public async markL1BatchSubmittedToL2(batchNumber: number): Promise<void> {
    return this.rdb.execute(
      `UPDATE l1_tx_batch
      SET status = 'SUBMITTED_TO_L2'
      WHERE batch_number = ${batchNumber}`
    )
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

  /**
   * @inheritDoc
   */
  public async tryBuildL2OnlyBatch(): Promise<number> {
    const timestampRes = await this.rdb.select(
      `SELECT DISTINCT block_timestamp
            FROM l2_tx
            WHERE status = '${L2BatchStatus.UNBATCHED}'
            ORDER BY block_timestamp ASC
      `
    )

    if (!timestampRes || timestampRes.length < 2) {
      return -1
    }

    const batchTimestamp = timestampRes[0].columns['block_timestamp']

    await this.rdb.startTransaction()
    try {
      const batchNumber = await this.insertNewL2TransactionBatch()
      await this.rdb.execute(`
        UPDATE l2_tx
        SET status = '${L2BatchStatus.BATCHED}', batch_number = ${batchNumber}
        WHERE status = '${L2BatchStatus.UNBATCHED}' AND block_timestamp = ${batchTimestamp}
      `)

      await this.rdb.commit()
      return batchNumber
    } catch (e) {
      logError(log, `Error building L2 Batch!`, e)
      await this.rdb.rollback()
      throw Error(e)
    }
  }

  public async tryBuildL2BatchToMatchL1(
    l1BatchSize: number,
    l1BatchNumber: number
  ): Promise<number> {
    const maxL2BatchNumber = await this.getMaxL2TxBatchNumber()
    if (maxL2BatchNumber >= l1BatchNumber) {
      log.debug(
        `Not attempting to build batch because max L2 batch number is ${maxL2BatchNumber} and provided L1 batchNumber is ${l1BatchNumber}`
      )
      return -1
    }

    const transactionsToBatchRes = await this.rdb.select(`
      SELECT COUNT(*) as batchable_tx_count, block_timestamp
      FROM l2_tx
      WHERE status = '${L2BatchStatus.UNBATCHED}'
      GROUP BY block_timestamp
      ORDER BY block_timestamp ASC
    `)

    if (
      !transactionsToBatchRes ||
      !transactionsToBatchRes.length ||
      !transactionsToBatchRes[0].columns['batchable_tx_count']
    ) {
      return -1
    }

    const batchableTxCount =
      transactionsToBatchRes[0].columns['batchable_tx-count']
    if (batchableTxCount < l1BatchSize && transactionsToBatchRes.length > 1) {
      const msg = `L2 transactions do not match L1 transactions! Cannot and will not be able to build an L2 batch until this is fixed! Expected L1 batch size ${l1BatchSize}, got multiple L2 batches with the oldest unbatched being of size ${batchableTxCount}`
      log.error(msg)
      throw Error(msg)
    }

    if (batchableTxCount < l1BatchSize) {
      return -1
    }

    await this.rdb.startTransaction()
    try {
      const batchNumber = await this.insertNewL2TransactionBatch()
      if (batchNumber !== l1BatchNumber) {
        log.error(
          `Created L2 batch number ${batchNumber} does not match expected L1 batch number ${l1BatchNumber}. This probably shouldn't happen.`
        )
        await this.rdb.rollback()
        return -1
      }
      await this.rdb.execute(`
        UPDATE l2_tx l
        SET l.status = '${L2BatchStatus.BATCHED}', l.batch_number = ${batchNumber}
        FROM (
            SELECT *
            FROM l2_tx
            WHERE status = '${L2BatchStatus.UNBATCHED}'
            LIMIT ${l1BatchSize}
        ) t
        WHERE l.id = t.id
      `)
      await this.rdb.commit()
      return batchNumber
    } catch (e) {
      logError(
        log,
        `Error creating L2 batch to match L1 batch of size ${l1BatchSize}.`,
        e
      )
      await this.rdb.rollback()
      throw Error(e)
    }
  }

  /**
   * @inheritDoc
   */
  public async getNextBatchForL1Submission(): Promise<L1BatchSubmission> {
    const res = await this.rdb.select(`
      SELECT b.batch_number, b.status, b.tx_batch_tx_hash, b.state_batch_tx_hash, tx.block_number, tx.block_timestamp, tx.tx_index, tx.tx_hash, tx.sender, tx.l1_message_sender, tx.target, tx.calldata, tx.nonce, tx.signature, tx.state_root
      FROM l2_tx tx
        INNER JOIN l2_tx_batch b 
      WHERE b.status = '${L2BatchStatus.BATCHED}'
      ORDER BY block_number ASC, tx_index ASC
    `)

    if (!res || !res.length || !res[0].data.length) {
      return undefined
    }

    const batch: L1BatchSubmission = {
      l1TxBatchTxHash: res[0].columns['tx_batch_tx_hash'],
      l1StateRootBatchTxHash: res[0].columns['state_batch_tx_hash'],
      status: res[0].columns['status'],
      l2BatchNumber: res[0].columns['batch_number'],
      transactions: [],
    }
    for (const row of res) {
      batch.transactions.push({
        timestamp: row.columns['block_timestamp'],
        blockNumber: row.columns['block_number'],
        transactionIndex: row.columns['tx_index'],
        transactionHash: row.columns['tx_hash'],
        to: row.columns['target'],
        from: row.columns['sender'],
        nonce: row.columns['nonce'],
        calldata: row.columns['calldata'],
        stateRoot: row.columns['state_root'],
        gasPrice: row.columns['gas_price'],
        gasLimit: row.columns['gas_limit'],
        l1MessageSender: row.columns['l1_message_sender'], // should never be present in this case
        signature: row.columns['signature'],
      })
    }

    return batch
  }

  /**
   * @inheritDoc
   */
  public async markTransactionBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    return this.rdb.execute(`
      UPDATE l2_tx_batch
      SET status = '${L2BatchStatus.TXS_SUBMITTED}', tx_batch_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}
    `)
  }

  /**
   * @inheritDoc
   */
  public async markTransactionBatchConfirmedOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    return this.rdb.execute(`
      UPDATE l2_tx_batch
      SET status = '${L2BatchStatus.TXS_CONFIRMED}', tx_batch_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}
    `)
  }

  /**
   * @inheritDoc
   */
  public async markStateRootBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    return this.rdb.execute(`
      UPDATE l2_tx_batch
      SET status = '${L2BatchStatus.ROOTS_SUBMITTED}', state_batch_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}
    `)
  }

  /**
   * @inheritDoc
   */
  public async markStateRootBatchConfirmedOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    return this.rdb.execute(`
      UPDATE l2_tx_batch 
      SET status = '${L2BatchStatus.ROOTS_CONFIRMED}', state_batch_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}
    `)
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
      FROM next_l1_verification_batch l1
        LEFT OUTER JOIN next_l2_verification_batch l2 
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
   * @inheritDoc
   */
  protected async insertNewL2TransactionBatch(): Promise<number> {
    let batchNumber: number

    let retries = 3
    // This should never fail, but adding in retries anyway
    while (retries > 0) {
      try {
        batchNumber = (await this.getMaxL2TxBatchNumber()) + 1
        await this.rdb.execute(`
            INSERT INTO l2_tx_batch(batch_number) 
            VALUES (${batchNumber})`)
        break
      } catch (e) {
        retries--
      }
    }

    return batchNumber
  }

  /**
   * Fetches the max L2 tx batch number for use in inserting a new tx batch
   * @returns The max batch number at the time of this query.
   */
  protected async getMaxL2TxBatchNumber(): Promise<number> {
    const rows = await this.rdb.select(
      `SELECT MAX(batch_number) as batch_number 
        FROM l2_tx_batch`
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
