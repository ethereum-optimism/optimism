/* External Imports */
import { DB } from '@pigi/core-db'
import {
  BigNumber,
  DefaultSignatureVerifier,
  getLogger,
  JsonRpcRequest,
  JsonRpcResponse,
  ONE,
  SignatureVerifier,
} from '@pigi/core-utils'
import {
  RollupStateMachine,
  SignedTransaction,
  TransactionResult,
} from '@pigi/rollup-core'

import AsyncLock from 'async-lock'
import FastPriorityQueue from 'fastpriorityqueue'

/* Internal Imports */
import { Aggregator, RollupBlockBuilder } from '../types'

const log = getLogger('aggregator')

export class DefaultAggregator implements Aggregator {
  public static readonly NEXT_TX_NUMBER_KEY = Buffer.from('next_tx_num')
  private static readonly lock_key = 'lock'

  private readonly lock: AsyncLock
  private readonly transactionResultQueue: FastPriorityQueue<TransactionResult>

  private nextTransactionToProcess: BigNumber

  public static async create(
    db: DB,
    stateMachine: RollupStateMachine,
    blockBuilder: RollupBlockBuilder,
    signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance()
  ): Promise<DefaultAggregator> {
    const aggregator: DefaultAggregator = new DefaultAggregator(
      db,
      stateMachine,
      blockBuilder,
      signatureVerifier
    )
    await aggregator.init()

    return aggregator
  }

  private constructor(
    private readonly db: DB,
    private readonly stateMachine: RollupStateMachine,
    private readonly blockBuilder: RollupBlockBuilder,
    private readonly signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance()
  ) {
    this.lock = new AsyncLock()
    this.transactionResultQueue = new FastPriorityQueue<TransactionResult>(
      (a: TransactionResult, b: TransactionResult) =>
        a.transactionNumber.lt(b.transactionNumber)
    )

    this.nextTransactionToProcess = ONE
  }

  /**
   * Gets any unprocessed transaction results that may exist sends them to BlockBuilder
   * to make sure all previous transactions are accounted for before handling new ones.
   */
  private async init(): Promise<void> {
    const nextTx: Buffer = await this.db.get(
      DefaultAggregator.NEXT_TX_NUMBER_KEY
    )
    if (!nextTx) {
      log.info(`No stored next transaction to process. Starting fresh.`)
      return
    }

    this.nextTransactionToProcess = new BigNumber(nextTx)

    const results: TransactionResult[] = await this.stateMachine.getTransactionResultsSince(
      this.nextTransactionToProcess.sub(ONE)
    )

    for (const res of results) {
      this.transactionResultQueue.add(res)
    }

    return this.processTransactionResultQueue()
  }

  public async handleRequest(
    request: JsonRpcRequest
  ): Promise<JsonRpcResponse> {
    // TODO: Forward to state machine, if tx, grab result and send it to block builder.
    return undefined
  }

  /**
   * Handles a SignedTransaction, processing it and returning a transaction receipt to the caller.
   *
   * @param signedTransaction The signed transaction to process.
   * @returns A signed transaction receipt.
   */
  public async handleTransaction(
    signedTransaction: SignedTransaction
  ): Promise<void> {
    const result: TransactionResult = await this.stateMachine.applyTransaction(
      signedTransaction
    )

    // Queued since multiple async calls may be returned out of order
    this.transactionResultQueue.add(result)
    return this.processTransactionResultQueue()

    // TODO: Create and return receipt
  }

  /**
   * Processes the transaction results queue in order, sending them to the Block Builder.
   */
  private async processTransactionResultQueue(): Promise<void> {
    return this.lock.acquire(DefaultAggregator.lock_key, async () => {
      while (
        this.transactionResultQueue.peek() &&
        this.transactionResultQueue
          .peek()
          .transactionNumber.eq(this.nextTransactionToProcess)
      ) {
        const res: TransactionResult = this.transactionResultQueue.poll()
        await this.blockBuilder.addTransactionResult(res)
        await this.db.put(
          DefaultAggregator.NEXT_TX_NUMBER_KEY,
          res.transactionNumber.toBuffer()
        )
        this.nextTransactionToProcess = res.transactionNumber.add(ONE)
      }
    })
  }
}
