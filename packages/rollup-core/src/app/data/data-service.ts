/* External Imports */
import { RDB, Row } from '@eth-optimism/core-db'
import {
  add0x,
  getLogger,
  logError,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'

import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import {
  GethSubmission,
  DataService,
  GethSubmissionQueueStatus,
  TransactionBatchSubmission,
  BatchSubmissionStatus,
  RollupTransaction,
  TransactionOutput,
  VerificationCandidate,
  VerificationStatus,
  StateCommitmentBatchSubmission,
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
    processed: boolean = false,
    txContext?: any
  ): Promise<void> {
    return this.rdb.execute(
      `${l1BlockInsertStatement} VALUES (${getL1BlockInsertValue(
        block,
        processed
      )})`,
      txContext
    )
  }

  /**
   * @inheritDoc
   */
  public async insertL1Transactions(
    transactions: TransactionResponse[],
    txContext?: any
  ): Promise<void> {
    if (!transactions || !transactions.length) {
      return
    }
    const values: string[] = transactions.map(
      (tx, index) => `(${getL1TransactionInsertValue(tx, index)})`
    )
    return this.rdb.execute(
      `${l1TxInsertStatement} VALUES ${values.join(',')}`,
      txContext
    )
  }

  /**
   * @inheritDoc
   */
  public async insertL1BlockAndTransactions(
    block: Block,
    txs: TransactionResponse[],
    processed: boolean = false
  ): Promise<void> {
    const txContext = await this.rdb.startTransaction()
    try {
      await this.insertL1Block(block, processed, txContext)
      await this.insertL1Transactions(txs, txContext)
    } catch (e) {
      await this.rdb.rollback(txContext)
      throw e
    }
    return this.rdb.commit(txContext)
  }

  /**
   * @inheritDoc
   */
  public async insertL1RollupTransactions(
    l1TxHash: string,
    rollupTransactions: RollupTransaction[],
    queueForGethSubmission: boolean = false
  ): Promise<number> {
    if (!rollupTransactions || !rollupTransactions.length) {
      return -1
    }

    let batchNumber: number
    const txContext = await this.rdb.startTransaction()
    try {
      if (queueForGethSubmission) {
        batchNumber = await this.insertGethSubmissionQueueEntry(
          rollupTransactions[0].l1TxHash,
          txContext
        )
      }

      const values: string[] = rollupTransactions.map(
        (x) => `(${getL1RollupTransactionInsertValue(x, batchNumber)})`
      )

      await this.rdb.execute(
        `${l1RollupTxInsertStatement} VALUES ${values.join(',')}`,
        txContext
      )

      await this.rdb.commit(txContext)
      return batchNumber !== undefined ? batchNumber : -1
    } catch (e) {
      logError(
        log,
        `Error inserting rollup tx batch #${batchNumber}, l1 Tx Hash: ${l1TxHash}, batch: ${JSON.stringify(
          rollupTransactions
        )}!`,
        e
      )
      await this.rdb.rollback(txContext)
      throw e
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
      FROM unqueued_rollup_tx
      WHERE queue_origin IN (${queueOrigins.join(',')}) 
      ORDER BY block_number ASC, l1_tx_index ASC, l1_tx_log_index ASC 
      LIMIT 1
    `)
    if (!txHashRes || !txHashRes.length || !txHashRes[0]['l1_tx_hash']) {
      return -1
    }
    log.debug(`Next Geth Submission res: ${JSON.stringify(txHashRes)}`)

    const txHash = txHashRes[0]['l1_tx_hash']
    const txLogIndex = txHashRes[0]['l1_tx_log_index']
    const queueOrigin = txHashRes[0]['queue_origin']

    const txContext = await this.rdb.startTransaction()
    try {
      const submissionQueueIndex: number = await this.insertGethSubmissionQueueEntry(
        txHash,
        txContext
      )

      await this.rdb.execute(
        `UPDATE l1_rollup_tx
        SET geth_submission_queue_index = ${submissionQueueIndex}, index_within_submission = 0
        WHERE l1_tx_hash = '${txHash}' AND l1_tx_log_index = ${txLogIndex}`,
        txContext
      )
      await this.rdb.commit(txContext)
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
      await this.rdb.rollback(txContext)
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
      return -1
    }

    let batchNumber
    const txContext = await this.rdb.startTransaction()
    try {
      batchNumber = await this.insertNewL1StateRootBatch(l1TxHash, txContext)

      const values: string[] = stateRoots.map(
        (root, i) =>
          `(${getL1RollupStateRootInsertValue(root, batchNumber, i)})`
      )
      await this.rdb.execute(
        `${l1RollupStateRootInsertStatement} VALUES ${values.join(',')}`,
        txContext
      )

      await this.rdb.commit(txContext)
      return batchNumber
    } catch (e) {
      logError(
        log,
        `Error inserting rollup state root batch #${batchNumber}, l1TxHash: ${l1TxHash}!`,
        e
      )
      await this.rdb.rollback(txContext)
      throw e
    }
  }

  /**
   * @inheritDoc
   */
  public async getNextQueuedGethSubmission(): Promise<GethSubmission> {
    const res: Row[] = await this.rdb.select(`
      SELECT geth_submission_queue_index, target, calldata, block_timestamp, block_number, l1_tx_hash, l1_tx_index, l1_tx_log_index, queue_origin, sender, l1_message_sender, gas_limit, nonce, signature
      FROM next_queued_geth_submission
    `)

    if (!res || !res.length) {
      return undefined
    }

    const gethSubmissionNumber = res[0]['geth_submission_queue_index']
    const timestamp = res[0]['block_timestamp']
    const blockNumber = res[0]['block_number']

    return {
      submissionNumber: gethSubmissionNumber,
      timestamp,
      blockNumber,
      rollupTransactions: res.map((row: Row, indexWithinSubmission: number) => {
        const tx: RollupTransaction = {
          l1RollupTxId: parseInt(row['id'], 10),
          indexWithinSubmission,
          l1TxHash: row['l1_tx_hash'],
          l1TxIndex: row['l1_tx_index'],
          l1TxLogIndex: row['l1_tx_log_index'],
          target: row['target'],
          calldata: row['calldata'], // TODO: may have to format Buffer => string
          l1Timestamp: row['block_timestamp'],
          l1BlockNumber: row['block_number'],
          queueOrigin: row['queue_origin'],
        }

        if (!!row['sender']) {
          tx.sender = row['sender']
        }
        if (!!row['l1MessageSender']) {
          tx.l1MessageSender = row['l1_message_sender']
        }
        if (!!row['gas_limit']) {
          tx.gasLimit = row['gas_limit']
        }
        if (!!row['nonce']) {
          tx.nonce = parseInt(row['nonce'], 10)
        }
        if (!!row['signature']) {
          tx.nonce = row['signature']
        }
        return tx
      }),
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
    WHERE block_hash = '${add0x(blockHash)}'`)
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
  public async tryBuildCanonicalChainBatchNotPresentOnL1(
    minBatchSize: number,
    maxBatchSize: number
  ): Promise<number> {
    // TODO: ADD SOME SIZE LIMIT
    const txRes = await this.rdb.select(
      `SELECT DISTINCT COUNT(*) as total, block_timestamp
            FROM l2_tx_output
            WHERE
              canonical_chain_batch_number IS NULL
              AND l1_rollup_tx_id IS NULL
            GROUP BY block_timestamp
            ORDER BY block_timestamp ASC
            LIMIT 2
      `
    )

    if (
      !txRes ||
      !txRes.length ||
      (txRes.length < 2 && parseInt(txRes[0]['total'], 10) < minBatchSize)
    ) {
      return -1
    }

    const batchTimestamp = parseInt(txRes[0]['block_timestamp'], 10)

    const txContext = await this.rdb.startTransaction()
    try {
      const batchNumber = await this.insertNewCanonicalChainBatch(txContext)
      await this.rdb.execute(
        `UPDATE l2_tx_output tx
        SET 
            canonical_chain_batch_number = ${batchNumber},
            canonical_chain_batch_index = t.row_number
        FROM 
          (
            SELECT id, row_number() over (ORDER BY id) -1 as row_number
            FROM l2_tx_output
            WHERE
              canonical_chain_batch_number IS NULL
              AND l1_rollup_tx_id IS NULL
              AND block_timestamp = ${batchTimestamp}
            ORDER BY block_number ASC, tx_index ASC
            LIMIT ${maxBatchSize}
          ) t
        WHERE tx.id = t.id 
          `,
        txContext
      )

      await this.rdb.commit(txContext)
      return batchNumber
    } catch (e) {
      logError(log, `Error building L2 Batch!`, e)
      await this.rdb.rollback(txContext)
      throw Error(e)
    }
  }

  /**
   * Determines whether or not the next State Commitment Chain batch represents a set of
   * state roots that were already appended to the L1 chain.
   *
   * @returns true if the next batch to build was already appended, false otherwise.
   */
  public async isNextStateCommitmentChainBatchToBuildAlreadyAppendedOnL1(): Promise<
    boolean
  > {
    const res = await this.rdb.select(
      `SELECT (l1.batch_number - l2.batch_number) as l1_lead
      FROM 
        (
          SELECT COALESCE(NULLIF(MAX(batch_number), 0), COUNT(*)) as batch_number
          FROM l1_rollup_state_root_batch
        ) l1,
        (
          SELECT COALESCE(NULLIF(MAX(batch_number), 0), COUNT(*)) as batch_number
          FROM state_commitment_chain_batch
          WHERE status <> '${BatchSubmissionStatus.QUEUED}'
        ) l2

      `
    )

    if (!res || !res.length || res[0]['l1_lead'] === undefined) {
      const msg = `Error performing isNextRootBatchFromL1 fetch! Returned an undefined result -- this should never happen!`
      log.error(msg)
      throw Error(msg)
    }

    return res[0]['l1_lead'] > 0
  }

  /**
   * @inheritDoc
   */
  public async tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch(): Promise<
    number
  > {
    const nextBatchNumber =
      (await this.getMaxStateCommitmentChainBatchNumber()) + 1
    const batchSizeRes = await this.rdb.select(
      `SELECT l1.batch_size as l1_batch_size, l2.batch_size as l2_batch_size
        FROM 
          (
            SELECT COUNT(*) as batch_size
            FROM l1_rollup_state_root
            WHERE batch_number = ${nextBatchNumber} 
          ) l1,
          (
            SELECT COUNT(*) as batch_size
            FROM l2_tx_output
            WHERE 
              state_commitment_chain_batch_number IS NULL
          ) l2  
      `
    )

    if (
      !batchSizeRes ||
      !batchSizeRes.length ||
      batchSizeRes[0]['l1_batch_size'] === undefined
    ) {
      const msg = `Unable to query L1 and L2 batch sizes for L1 Batch Number ${nextBatchNumber}`
      log.error(msg)
      throw Error(msg)
    }

    const l1BatchSize: number = parseInt(batchSizeRes[0]['l1_batch_size'], 10)
    if (l1BatchSize === 0 || l1BatchSize > batchSizeRes[0]['l2_batch_size']) {
      log.debug(
        `Cannot build L2 state commitment batch to match L1 batch number ${nextBatchNumber} yet because there are ${l1BatchSize} roots in the L1 batch and ${batchSizeRes['l2_batch_size']} processed L2 roots.`
      )
      return -1
    }

    const txContext = await this.rdb.startTransaction()
    try {
      const batchNumber = await this.insertNewStateCommitmentChainBatch(
        true,
        txContext
      )
      if (batchNumber !== nextBatchNumber) {
        log.error(
          `Created L2 batch number ${batchNumber} does not match expected batch number ${nextBatchNumber}. This probably shouldn't happen.`
        )
        await this.rdb.rollback(txContext)
        return -1
      }
      await this.rdb.execute(
        `UPDATE l2_tx_output as tx
        SET 
            state_commitment_chain_batch_number = ${batchNumber},
            state_commitment_chain_batch_index = t.row_number
        FROM (
          SELECT id, row_number() over (ORDER BY id) -1 as row_number
          FROM l2_tx_output
          WHERE state_commitment_chain_batch_number IS NULL
          ORDER BY block_number ASC, tx_index ASC
          LIMIT ${l1BatchSize}
        ) t
        WHERE tx.id = t.id`,
        txContext
      )
      await this.rdb.commit(txContext)
      return batchNumber
    } catch (e) {
      logError(
        log,
        `Error creating State Commitment Chain batch to match L1 batch of size ${l1BatchSize}.`,
        e
      )
      await this.rdb.rollback(txContext)
      throw Error(e)
    }
  }

  /**
   * @inheritDoc
   */
  public async tryBuildL2OnlyStateCommitmentChainBatch(
    minBatchSize: number,
    maxBatchSize: number
  ): Promise<number> {
    const availableRootsRes = await this.rdb.select(
      `SELECT COUNT(*) as available
      FROM batchable_l2_only_tx_states`
    )

    if (
      !availableRootsRes ||
      !availableRootsRes.length ||
      availableRootsRes[0]['available'] === undefined
    ) {
      const msg = `Error: unable to fetch available L2 Tx Output State Roots`
      log.error(msg)
      throw Error(msg)
    }

    if (parseInt(availableRootsRes[0]['available'], 10) < minBatchSize) {
      log.debug(
        `Cannot build L2-only state commitment batch. Only ${availableRootsRes[0]['available']} unbatched L2 roots exist`
      )
      return -1
    }

    const txContext = await this.rdb.startTransaction()
    try {
      const batchNumber = await this.insertNewStateCommitmentChainBatch(
        false,
        txContext
      )
      await this.rdb.execute(
        `UPDATE l2_tx_output as tx
        SET 
            state_commitment_chain_batch_number = ${batchNumber},
            state_commitment_chain_batch_index = t.row_number
        FROM (
          SELECT id, row_number
          FROM batchable_l2_only_tx_states
          ORDER BY block_number ASC, tx_index ASC
          LIMIT ${maxBatchSize}
        ) t
        WHERE tx.id = t.id`,
        txContext
      )
      await this.rdb.commit(txContext)
      return batchNumber
    } catch (e) {
      logError(log, `Error creating L2-only State Commitment Chain.`, e)
      await this.rdb.rollback(txContext)
      throw Error(e)
    }
  }

  /**
   * @inheritDoc
   */
  public async getNextCanonicalChainTransactionBatchToSubmit(): Promise<
    TransactionBatchSubmission
  > {
    const res = await this.rdb.select(
      `SELECT cc.batch_number, cc.status, cc.submission_tx_hash, tx.block_number, tx.block_timestamp, tx.tx_index, tx.tx_hash, tx.sender, tx.l1_message_sender, tx.target, tx.calldata, tx.nonce, tx.signature, tx.state_root
      FROM l2_tx_output tx
        INNER JOIN canonical_chain_batch cc ON tx.canonical_chain_batch_number = cc.batch_number 
      WHERE batch_number = (
            SELECT MIN(batch_number)
            FROM canonical_chain_batch
            WHERE status = '${BatchSubmissionStatus.QUEUED}'
          )
      ORDER BY block_number ASC, tx_index ASC`
    )

    if (!res || !res.length || !res[0]) {
      log.debug(`No canonical chain tx batch to submit.`)
      return undefined
    }

    const batch: TransactionBatchSubmission = {
      submissionTxHash: res[0]['submission_tx_hash'],
      status: res[0]['status'],
      batchNumber: res[0]['batch_number'],
      transactions: [],
    }
    for (const row of res) {
      batch.transactions.push({
        timestamp: row['block_timestamp'],
        blockNumber: row['block_number'],
        transactionIndex: row['tx_index'],
        transactionHash: row['tx_hash'],
        to: row['target'] || ZERO_ADDRESS,
        from: row['sender'],
        nonce: parseInt(row['nonce'], 10),
        calldata: row['calldata'],
        stateRoot: row['state_root'],
        gasPrice: row['gas_price'],
        gasLimit: row['gas_limit'],
        l1MessageSender: row['l1_message_sender'] || undefined, // should never be present in this case
        signature: row['signature'],
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
      `UPDATE canonical_chain_batch
      SET status = '${BatchSubmissionStatus.SENT}', submission_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}`
    )
  }

  /**
   * @inheritDoc
   */
  public async markTransactionBatchFinalOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    return this.rdb.execute(
      `UPDATE canonical_chain_batch
      SET status = '${BatchSubmissionStatus.FINALIZED}', submission_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}`
    )
  }

  public async getNextStateCommitmentBatchToSubmit(): Promise<
    StateCommitmentBatchSubmission
  > {
    const res = await this.rdb.select(
      `SELECT scc.batch_number, scc.status, scc.submission_tx_hash, tx.state_root
      FROM l2_tx_output tx
        INNER JOIN state_commitment_chain_batch scc ON tx.state_commitment_chain_batch_number = scc.batch_number 
      WHERE batch_number = (
            SELECT MIN(batch_number)
            FROM state_commitment_chain_batch
            WHERE status = '${BatchSubmissionStatus.QUEUED}'
          )
      ORDER BY block_number ASC, tx_index ASC`
    )

    if (!res || !res.length || !res[0]) {
      return undefined
    }

    return {
      submissionTxHash: res[0]['submission_tx_hash'],
      status: res[0]['status'],
      batchNumber: res[0]['batch_number'],
      stateRoots: res.map((x: Row) => x['state_root']),
    }
  }

  /**
   * @inheritDoc
   */
  public async markStateRootBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    return this.rdb.execute(
      `UPDATE state_commitment_chain_batch
      SET status = '${BatchSubmissionStatus.SENT}', submission_tx_hash = '${l1TxHash}'
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
      `UPDATE state_commitment_chain_batch 
      SET status = '${BatchSubmissionStatus.FINALIZED}', submission_tx_hash = '${l1TxHash}'
      WHERE batch_number = ${batchNumber}`
    )
  }

  /************
   * VERIFIER *
   ************/

  /**
   * @inheritDoc
   */
  public async getNextVerificationCandidate(): Promise<VerificationCandidate> {
    const rows: Row[] = await this.rdb.select(
      `SELECT batch_number, batch_index, l1_root, geth_root
      FROM next_verification_batch`
    )

    if (!rows || !rows.length) {
      return undefined
    }

    if (!rows[rows.length - 1]['geth_root']) {
      // No L2 root has been calculated for the last item in the batch -- cannot yet verify.
      return undefined
    }

    return {
      batchNumber: rows[0]['batch_number'],
      roots: rows.map((x) => {
        return {
          l1Root: x['l1_root'],
          gethRoot: x['geth_root'],
        }
      }),
    }
  }

  /**
   * @inheritDoc
   */
  public async verifyStateRootBatch(batchNumber: number): Promise<void> {
    await this.rdb.execute(
      `UPDATE l1_rollup_state_root_batch
      SET status = '${VerificationStatus.VERIFIED}'
      WHERE batch_number = ${batchNumber}
        AND status = '${VerificationStatus.UNVERIFIED}'`
    )
  }

  /**
   * @inheritDoc
   */
  public async markVerificationCandidateFraudulent(
    batchNumber: number
  ): Promise<void> {
    await this.rdb.execute(
      `UPDATE l1_rollup_state_root_batch
      SET status = '${VerificationStatus.FRAUDULENT}'
      WHERE batch_number = ${batchNumber}
        AND status = '${VerificationStatus.UNVERIFIED}'`
    )
  }

  /***********
   * HELPERS *
   ***********/

  /**
   * Inserts a new Geth Submission Queue entry with an index one higher than the previous one.
   *
   * @param l1TxHash The L1 tx hash from which the txs in this entry came.
   * @param txContext The tx context if there is one.
   * @returns The new entry's index in the queue.
   */
  protected async insertGethSubmissionQueueEntry(
    l1TxHash: string,
    txContext?: any
  ): Promise<number> {
    let queueIndex: number

    let retries = 3
    // This should never fail, but adding in retries anyway
    while (retries > 0) {
      try {
        queueIndex = (await this.getMaxGethSubmissionQueueIndex()) + 1
        await this.rdb.execute(
          `INSERT INTO geth_submission_queue(l1_tx_hash, queue_index) 
            VALUES ('${l1TxHash}', ${queueIndex})`,
          txContext
        )
        break
      } catch (e) {
        retries--
      }
    }

    return queueIndex
  }

  /**
   * Inserts a new L1 State Root batch with a batch number one higher than the previous one.
   *
   * @param l1TxHash The L1 tx hash from which this state root batch came.
   * @param txContext The tx context to use for this insert, if any.
   * @returns The new entry's batch number.
   */
  protected async insertNewL1StateRootBatch(
    l1TxHash: string,
    txContext?: any
  ): Promise<number> {
    let batchNumber: number

    let retries = 3
    // This should never fail, but adding in retries anyway
    while (retries > 0) {
      try {
        batchNumber = (await this.getMaxL1StateRootBatchNumber()) + 1
        await this.rdb.execute(
          `INSERT INTO l1_rollup_state_root_batch(l1_tx_hash, batch_number) 
            VALUES ('${l1TxHash}', ${batchNumber})`,
          txContext
        )
        return batchNumber
      } catch (e) {
        retries--
        if (retries === 0) {
          logError(log, `Error inserting new L1 State Root Batch`, e)
          throw e
        }
      }
    }
  }

  /**
   * Inserts a new Canonical Chain Tx batch with a batch number one higher than the previous one.
   *
   * @param txContext The tx context to use for this insert, if there is one.
   * @returns The new entry's batch number.
   */
  protected async insertNewCanonicalChainBatch(
    txContext?: any
  ): Promise<number> {
    let batchNumber: number

    let retries = 3
    // This should never fail, but adding in retries anyway
    while (retries > 0) {
      try {
        batchNumber =
          (await this.getMaxCanonicalChainBatchNumber(txContext)) + 1
        await this.rdb.execute(
          `INSERT INTO canonical_chain_batch(batch_number) 
            VALUES (${batchNumber})`,
          txContext
        )
        return batchNumber
      } catch (e) {
        retries--
        if (retries === 0) {
          logError(log, `Error inserting new canonical chain batch`, e)
          throw e
        }
      }
    }
  }

  /**
   * Fetches the max Canonical Chain tx batch number for use in inserting a new tx batch.
   *
   * @param txContext The tx context to use for this fetch if any.
   * @returns The max batch number at the time of this query.
   */
  protected async getMaxCanonicalChainBatchNumber(
    txContext?: any
  ): Promise<number> {
    const rows = await this.rdb.select(
      `SELECT MAX(batch_number) as batch_number
         FROM canonical_chain_batch`,
      txContext
    )
    if (
      rows &&
      !!rows.length &&
      !!rows[0] &&
      rows[0]['batch_number'] !== undefined &&
      rows[0]['batch_number'] !== null
    ) {
      return parseInt(rows[0]['batch_number'], 10)
    }

    return -1
  }

  /**
   * Inserts a new State Commitment batch and returns the inserted batch number.
   *
   * @param final Whether or not the batch should be created as final.
   * @param txContext The tx context for this insert, if any
   * @returns The created batch number.
   */
  protected async insertNewStateCommitmentChainBatch(
    final: boolean = false,
    txContext?: any
  ): Promise<number> {
    let batchNumber: number

    let retries = 3
    // This should never fail, but adding in retries anyway
    while (retries > 0) {
      try {
        batchNumber = (await this.getMaxStateCommitmentChainBatchNumber()) + 1
        await this.rdb.execute(
          `INSERT INTO state_commitment_chain_batch(batch_number, status) 
            VALUES (${batchNumber}, '${
            final
              ? BatchSubmissionStatus.FINALIZED
              : BatchSubmissionStatus.QUEUED
          }')`,
          txContext
        )
        return batchNumber
      } catch (e) {
        retries--
        if (retries === 0) {
          logError(log, `Error inserting new state commitment chain batch`, e)
          throw e
        }
      }
    }
  }

  /**
   * Fetches the max State Commitment Chain batch number for use in inserting a new root batch.
   * @returns The max batch number at the time of this query.
   */
  protected async getMaxStateCommitmentChainBatchNumber(): Promise<number> {
    const rows = await this.rdb.select(
      `SELECT MAX(batch_number) as batch_number 
        FROM state_commitment_chain_batch`
    )
    if (
      rows &&
      !!rows.length &&
      !!rows[0] &&
      rows[0]['batch_number'] !== undefined &&
      rows[0]['batch_number'] !== null
    ) {
      return parseInt(rows[0]['batch_number'], 10)
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
      !!rows[0] &&
      rows[0]['queue_index'] !== undefined &&
      rows[0]['queue_index'] !== null
    ) {
      return parseInt(rows[0]['queue_index'], 10)
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
        FROM l1_rollup_state_root_batch`
    )
    if (
      rows &&
      !!rows.length &&
      !!rows[0] &&
      rows[0]['batch_number'] !== undefined &&
      rows[0]['batch_number'] !== null
    ) {
      return parseInt(rows[0]['batch_number'], 10)
    }

    return -1
  }
}
