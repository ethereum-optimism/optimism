/* External Imports */
import { RDB, Row } from '@eth-optimism/core-db'
import { getLogger, logError } from '@eth-optimism/core-utils'

import { Block, TransactionResponse } from 'ethers/providers'
/* Internal Imports */
import {
  BlockBatches,
  DataService,
  GethSubmissionQueueStatus,
  GethSubmissionRecord,
  OccBatchSubmission,
  OptimisticCanonicalChainStatus,
  QueueOrigin,
  RollupTransaction,
  TransactionOutput,
  VerificationCandidate,
  VerificationStatus,
} from '../../types'
import {
  getL1BlockInsertValue,
  getL1RollupStateRootInsertValue,
  getL1RollupTransactionInsertValue,
  getL1TransactionInsertValue,
  getL2TransactionOutputInsertValue,
  l1BlockInsertStatement,
  l1RollupStateRootInsertStatement,
  l1RollupTxInsertStatement,
  l1TxInsertStatement,
  l2TransactionOutputInsertStatement,
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
        batchNumber = await this.insertGethSubmissionQueueEntry(
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
  public async queueNextGethSubmission(
    queueOrigins: number[]
  ): Promise<number> {
    const txHashRes = await this.rdb.select(`
      SELECT l1_tx_hash, l1_tx_log_index, queue_origin
      FROM unbatched_rollup_tx
      WHERE queue_origin IN (${queueOrigins.join(',')}) 
      ORDER BY l1_block_number ASC, l1_tx_index ASC, l1_tx_log_index ASC 
      LIMIT 1
    `)
    if (
      !txHashRes ||
      !txHashRes.length ||
      !txHashRes[0].columns['l1_tx_hash']
    ) {
      return undefined
    }

    const txHash = txHashRes[0].columns['l1_tx_hash']
    const txLogIndex = txHashRes[0].columns['l1_tx_log_index']
    const queueOrigin = txHashRes[0].columns['queue_origin']

    await this.rdb.startTransaction()
    try {
      const submissionQueueIndex: number = await this.insertGethSubmissionQueueEntry(
        txHash
      )

      await this.rdb.execute(`
        UPDATE l1_rollup_tx
        SET geth_submission_queue_index = ${submissionQueueIndex}, index_within_submission = 0
        WHERE l1_tx_hash = '${txHash}' AND l1_tx_log_index = ${txLogIndex} 
      `)

      log.debug(
        `Created Geth submission queue index ${submissionQueueIndex} and queue origin ${queueOrigin}`
      )
      return submissionQueueIndex
    } catch (e) {
      logError(
        log,
        `Error executing queueNextGethSubmission for tx hash ${txHash} and queue origin ${queueOrigin}... rolling back`,
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
  public async getOldestQueuedGethSubmission(): Promise<GethSubmissionRecord> {
    const res: Row[] = await this.rdb.select(`
      SELECT COUNT(*) as submission_size, geth_submission_queue_index, block_timestamp, 
      FROM next_queued_geth_submission
      GROUP BY geth_submission_queue_index, block_timestamp
      ORDER BY geth_submission_queue_index ASC
      LIMIT 1
    `)

    if (!res || !res.length || !res[0].columns['batch_size']) {
      return undefined
    }
    return {
      size: res[0].columns['submission_size'],
      submissionNumber: res[0].columns['geth_submission_queue_index'],
      blockTimestamp: res[0].columns['block_timestamp'],
    }
  }

  /**
   * @inheritDoc
   */
  public async getNextQueuedGethSubmission(): Promise<BlockBatches> {
    const res: Row[] = await this.rdb.select(`
      SELECT geth_submission_queue_index, target, calldata, block_timestamp, block_number, l1_tx_hash, l1_tx_index, l1_tx_log_index, queue_origin, sender, l1_message_sender, gas_limit, nonce, signature
      FROM next_queued_geth_submission
    `)

    if (!res || !res.length) {
      return undefined
    }

    const gethSubmissionNumber = res[0].columns['geth_submission_queue_index']
    const timestamp = res[0].columns['block_timestamp']
    const blockNumber = res[0].columns['block_number']

    return {
      batchNumber: gethSubmissionNumber,
      timestamp,
      blockNumber,
      batches: [
        res.map((row: Row, indexWithinSubmission: number) => {
          const tx: RollupTransaction = {
            indexWithinSubmission,
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
  public async markQueuedGethSubmissionSubmittedToGeth(
    queueIndex: number
  ): Promise<void> {
    return this.rdb.execute(
      `UPDATE geth_submission_queue
      SET status = '${GethSubmissionQueueStatus.SENT}'
      WHERE queue_index = ${queueIndex}`
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
  public async insertL2TransactionOutput(tx: TransactionOutput): Promise<void> {
    return this.rdb.execute(
      `${l2TransactionOutputInsertStatement} VALUES (${getL2TransactionOutputInsertValue(
        tx
      )})`
    )
  }

  /**
   * @inheritDoc
   */
  public async tryBuildOccBatchNotPresentOnL1(): Promise<number> {
    // TODO: ADD SOME SIZE LIMIT
    const timestampRes = await this.rdb.select(
      `SELECT DISTINCT block_timestamp
            FROM l2_tx_output
            WHERE occ_batch_number IS NULL
            ORDER BY block_timestamp ASC
            LIMIT 2
      `
    )

    if (!timestampRes || timestampRes.length < 2) {
      return -1
    }

    const batchTimestamp = timestampRes[0].columns['block_timestamp']

    await this.rdb.startTransaction()
    try {
      const batchNumber = await this.insertNewOccBatch()
      await this.rdb.execute(`
        UPDATE l2_tx_output
        SET status = '${OptimisticCanonicalChainStatus.QUEUED}', occ_batch_number = ${batchNumber}
        WHERE occ_batch_number IS NULL AND block_timestamp = ${batchTimestamp}
      `)

      await this.rdb.commit()
      return batchNumber
    } catch (e) {
      logError(log, `Error building L2 Batch!`, e)
      await this.rdb.rollback()
      throw Error(e)
    }
  }

  public async tryBuildOccBatchToMatchL1Batch(
    l1BatchSize: number,
    l1BatchNumber: number
  ): Promise<number> {
    const maxL2BatchNumber = await this.getMaxOccBatchNumber()
    if (maxL2BatchNumber >= l1BatchNumber) {
      log.debug(
        `Not attempting to build batch because max L2 batch number is ${maxL2BatchNumber} and provided L1 batchNumber is ${l1BatchNumber}`
      )
      return -1
    }

    const transactionsToBatchRes = await this.rdb.select(`
      SELECT COUNT(*) as batchable_tx_count, block_timestamp
      FROM l2_tx_output
      WHERE occ_batch_number IS NULL
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
      transactionsToBatchRes[0].columns['batchable_tx_count']
    if (batchableTxCount < l1BatchSize && transactionsToBatchRes.length > 1) {
      const msg = `L2 transactions do not match L1 transactions! Cannot and will not be able to build an OCC Batch until this is fixed! Expected L1 batch size ${l1BatchSize}, got multiple tx block timestamps with the oldest unbatched tx set being of size ${batchableTxCount}`
      log.error(msg)
      throw Error(msg)
    }

    if (batchableTxCount < l1BatchSize) {
      return -1
    }

    await this.rdb.startTransaction()
    try {
      const batchNumber = await this.insertNewOccBatch()
      if (batchNumber !== l1BatchNumber) {
        log.error(
          `Created L2 batch number ${batchNumber} does not match expected L1 batch number ${l1BatchNumber}. This probably shouldn't happen.`
        )
        await this.rdb.rollback()
        return -1
      }
      await this.rdb.execute(
        `UPDATE l2_tx_output l
        SET l.status = '${OptimisticCanonicalChainStatus.QUEUED}', l.occ_batch_number = ${batchNumber}
        FROM (
          SELECT *
          FROM l2_tx_output
          WHERE occ_batch_number IS NULL
          LIMIT ${l1BatchSize}
        ) t
        WHERE l.id = t.id`
      )
      await this.rdb.commit()
      return batchNumber
    } catch (e) {
      logError(
        log,
        `Error creating OCC batch to match L1 batch of size ${l1BatchSize}.`,
        e
      )
      await this.rdb.rollback()
      throw Error(e)
    }
  }

  /**
   * @inheritDoc
   */
  public async getNextOccTransactionBatchToSubmit(): Promise<
    OccBatchSubmission
  > {
    const res = await this.rdb.select(
      `SELECT occ.batch_number, occ.tx_batch_status, occ.root_batch_status, occ.submitted_tx_batch_l1_tx_hash, occ.submitted_root_batch_l1_tx_hash, tx.block_number, tx.block_timestamp, tx.tx_index, tx.tx_hash, tx.sender, tx.l1_message_sender, tx.target, tx.calldata, tx.nonce, tx.signature, tx.state_root
      FROM l2_tx tx
        INNER JOIN optimistic_canonical_chain_batch occ ON tx.batch_number = occ.batch_number 
      WHERE occ.tx_status = '${OptimisticCanonicalChainStatus.QUEUED}'
      ORDER BY block_number ASC, tx_index ASC`
    )

    if (!res || !res.length || !res[0].data.length) {
      return undefined
    }

    const batch: OccBatchSubmission = {
      l1TxBatchTxHash: res[0].columns['submitted_tx_batch_l1_tx_hash'],
      l1StateRootBatchTxHash: res[0].columns['submitted_root_batch_l1_tx_hash'],
      txBatchStatus: res[0].columns['tx_batch_status'],
      rootBatchStatus: res[0].columns['root_batch_status'],
      occBatchNumber: res[0].columns['batch_number'],
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
    return this.rdb.execute(
      `UPDATE optimistic_canonical_chain_batch
      SET tx_batch_status = '${OptimisticCanonicalChainStatus.SENT}', submitted_tx_batch_l1_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}`
    )
  }

  /**
   * @inheritDoc
   */
  public async markTransactionBatchConfirmedOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    return this.rdb.execute(
      `UPDATE optimistic_canonical_chain_batch
      SET tx_batch_status = '${OptimisticCanonicalChainStatus.FINALIZED}', submitted_tx_batch_l1_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}`
    )
  }

  /**
   * @inheritDoc
   */
  public async markStateRootBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    return this.rdb.execute(
      `UPDATE optimistic_canonical_chain_batch
      SET root_batch_status = '${OptimisticCanonicalChainStatus.SENT}', submitted_root_batch_l1_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}`
    )
  }

  /**
   * @inheritDoc
   */
  public async markStateRootBatchFinalOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    return this.rdb.execute(
      `UPDATE optimistic_canonical_chain_batch 
      SET root_batch_status = '${OptimisticCanonicalChainStatus.FINALIZED}', submitted_root_batch_l1_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}`
    )
  }

  /************
   * VERIFIER *
   ************/

  /**
   * @inheritDoc
   */
  public async getVerificationCandidate(): Promise<VerificationCandidate> {
    const rows: Row[] = await this.rdb.select(
      `SELECT batch_number, l1_batch_index, l1_root, geth_root
      FROM next_verification_batch`
    )

    if (!rows || !rows.length) {
      return undefined
    }

    if (!rows[rows.length - 1].columns['geth_root']) {
      // No L2 root has been calculated for the last item in the batch -- cannot yet verify.
      return undefined
    }

    return {
      batchNumber: rows[0].columns['batch_number'],
      roots: rows.map((x) => {
        return {
          l1Root: x.columns['l1_root'],
          gethRoot: x.columns['geth_root'],
        }
      }),
    }
  }

  /**
   * @inheritDoc
   */
  public async verifyStateRootBatch(batchNumber): Promise<void> {
    await this.rdb.execute(
      `UPDATE l1_state_root_batch
      SET status = ${VerificationStatus.VERIFIED}
      WHERE batch_number = ${batchNumber}`
    )
  }

  /***********
   * HELPERS *
   ***********/

  /**
   * @inheritDoc
   */
  protected async insertGethSubmissionQueueEntry(
    l1TxHash: string
  ): Promise<number> {
    let queueIndex: number

    let retries = 3
    // This should never fail, but adding in retries anyway
    while (retries > 0) {
      try {
        queueIndex = (await this.getMaxGethSubmissionQueueIndex()) + 1
        await this.rdb.execute(
          `INSERT INTO geth_submission_queue(l1_tx_hash, queue_index) 
            VALUES ('${l1TxHash}', ${queueIndex})`
        )
        break
      } catch (e) {
        retries--
      }
    }

    return queueIndex
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
        await this.rdb.execute(
          `INSERT INTO l1_state_root_batch(l1_tx_hash, batch_number) 
            VALUES ('${l1TxHash}', ${batchNumber})`
        )
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
  protected async insertNewOccBatch(): Promise<number> {
    let batchNumber: number

    let retries = 3
    // This should never fail, but adding in retries anyway
    while (retries > 0) {
      try {
        batchNumber = (await this.getMaxOccBatchNumber()) + 1
        await this.rdb.execute(`
            INSERT INTO optimistic_canonical_chain_batch(batch_number) 
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
  protected async getMaxOccBatchNumber(): Promise<number> {
    const rows = await this.rdb.select(
      `SELECT MAX(batch_number) as batch_number 
        FROM optimistic_canonical_chain_batch`
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
  protected async getMaxGethSubmissionQueueIndex(): Promise<number> {
    const rows = await this.rdb.select(
      `SELECT MAX(queue_index) as queue_index 
        FROM geth_submission_queue`
    )
    if (
      rows &&
      !!rows.length &&
      !!rows[0].columns &&
      !!rows[0].columns['queue_index']
    ) {
      // TODO: make sure we don't need to cast
      return rows[0].columns['queue_index']
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
