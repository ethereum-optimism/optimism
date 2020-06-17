/* External Imports */
import {
  BaseQueuedPersistedProcessor,
  DB,
  EthereumListener,
} from '@eth-optimism/core-db'
import { getLogger, Logger } from '@eth-optimism/core-utils'

import { Block, Provider, TransactionResponse } from 'ethers/providers'
import { Log } from 'ethers/providers/abstract-provider'

/* Internal Imports */
import {
  BlockBatches,
  BlockBatchListener,
  BatchLogParserContext,
  L1Batch,
} from '../../types'
import { addressesAreEqual } from '../utils'

const log: Logger = getLogger('block-batch-processor')

export class BlockBatchProcessor
  extends BaseQueuedPersistedProcessor<BlockBatches>
  implements EthereumListener<Block> {
  public static readonly persistenceKey = 'BlockBatchProcessor'

  private readonly topics: string[]
  private readonly topicMap: Map<string, BatchLogParserContext>

  /**
   * Creates a BlockBatchProcessor that subscribes to blocks, processes all
   * L1ToL2Transaction events, parses L1ToL2Transactions and submits them to L2.
   *
   * @param db The DB to use to persist the queue of L1ToL2Transaction[] objects.
   * @param l1Provider The provider to use to connect to L1 to subscribe & fetch block / tx / log data.
   * @param logContexts The collection of L1ToL2TransactionLogParserContext that uniquely identify the log event and
   *        provide the ability to create L2 transactions from the L1 transaction that emitted it.
   * @param listeners The downstream subscribers to the L1ToL2TransactionBatch objects this processor creates.
   * @param persistenceKey The persistence key to use for this instance within the provided DB.
   */
  public static async create(
    db: DB,
    l1Provider: Provider,
    logContexts: BatchLogParserContext[],
    listeners: BlockBatchListener[],
    persistenceKey: string = BlockBatchProcessor.persistenceKey
  ): Promise<BlockBatchProcessor> {
    const processor = new BlockBatchProcessor(
      db,
      l1Provider,
      logContexts,
      listeners,
      persistenceKey
    )
    await processor.init()
    return processor
  }

  private constructor(
    db: DB,
    private readonly l1Provider: Provider,
    logContexts: BatchLogParserContext[],
    private readonly listeners: BlockBatchListener[],
    persistenceKey: string = BlockBatchProcessor.persistenceKey
  ) {
    super(db, persistenceKey)
    this.topicMap = new Map<string, BatchLogParserContext>(
      logContexts.map((x) => [x.topic, x])
    )
    this.topics = Array.from(this.topicMap.keys())
  }

  /**
   * @inheritDoc
   */
  public async handle(block: Block): Promise<void> {
    log.debug(
      `Received block ${block.number}. Searching for any contained L1toL2Transactions.`
    )

    const logs: Log[] = await this.l1Provider.getLogs({
      blockHash: block.hash,
      topics: this.topics,
    })
    log.debug(
      `Got ${logs.length} logs from block ${block.number}: ${JSON.stringify(
        logs
      )}`
    )

    logs.sort((a, b) => a.logIndex - b.logIndex)

    let batches: L1Batch[] = await Promise.all(
      logs.map((l) => this.getBatchFromLog(l))
    )

    batches = batches.filter((x) => x.length > 0)

    if (!batches.length) {
      log.debug(`There were no L1toL2Transactions in block ${block.number}.`)
    } else {
      log.debug(
        `Parsed ${batches.length} batches from block ${
          block.number
        }: ${JSON.stringify(batches)}`
      )
    }

    this.add(block.number, {
      blockNumber: block.number,
      timestamp: block.timestamp,
      batches,
    })
  }

  /**
   * @inheritDoc
   */
  public async onSyncCompleted(syncIdentifier?: string): Promise<void> {
    // TODO: Turn off processing of CannonicalTransactionChainBatch events here
  }

  /**
   * @inheritDoc
   */
  protected async handleNextItem(
    blockNumber: number,
    blockBatches: BlockBatches
  ): Promise<void> {
    try {
      if (!!blockBatches.batches.length) {
        this.listeners.map((x) => x.handleBlockBatches(blockBatches))
      }
      await this.markProcessed(blockNumber)
    } catch (e) {
      this.logError(
        `Error processing L1ToL2Transactions. Txs: ${JSON.stringify(
          blockBatches
        )}`,
        e
      )
      // Can't properly sync from L1 to L2, and need to do so in order. This is fatal.
      process.exit(1)
    }
  }

  /**
   * @inheritDoc
   */
  protected async serializeItem(item: BlockBatches): Promise<Buffer> {
    return Buffer.from(JSON.stringify(item), 'utf-8')
  }

  /**
   * @inheritDoc
   */
  protected async deserializeItem(itemBuffer: Buffer): Promise<BlockBatches> {
    return JSON.parse(itemBuffer.toString('utf-8'))
  }

  private async getBatchFromLog(l: Log): Promise<L1Batch> {
    const matchedTopics: string[] = l.topics.filter(
      (x) => this.topics.indexOf(x) >= 0
    )
    if (matchedTopics.length === 0) {
      log.error(
        `Received log with topics: ${l.topics.join(
          ','
        )} for subscription to topics: ${this.topics.join(',')}. Transaction: ${
          l.transactionHash
        }`
      )
      return []
    }

    const transaction: TransactionResponse = await this.l1Provider.getTransaction(
      l.transactionHash
    )
    log.debug(
      `Fetched tx by hash ${l.transactionHash}: ${JSON.stringify(transaction)}`
    )

    const parsedBatch: L1Batch = []
    for (const topic of matchedTopics) {
      const context = this.topicMap.get(topic)
      if (!addressesAreEqual(l.address, context.contractAddress)) {
        continue
      }
      const transactions = await context.parseL1Batch(l, transaction)
      parsedBatch.push(...transactions)
    }

    return parsedBatch
  }
}
