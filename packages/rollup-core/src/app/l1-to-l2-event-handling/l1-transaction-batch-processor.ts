/* External Imports */
import {
  BaseQueuedPersistedProcessor,
  DB,
  EthereumListener,
} from '@eth-optimism/core-db'
import { getLogger, Logger } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  L1ToL2Transaction,
  L1ToL2TransactionBatch,
  L1ToL2TransactionBatchListener,
  L1ToL2TransactionLogParserContext,
} from '../../types'
import { Block, Provider, TransactionResponse } from 'ethers/providers'
import { Log } from 'ethers/providers/abstract-provider'
import { addressesAreEqual } from '../utils'

const log: Logger = getLogger('l1-to-l2-transition-synchronizer')

export class L1TransactionBatchProcessor
  extends BaseQueuedPersistedProcessor<L1ToL2TransactionBatch>
  implements EthereumListener<Block> {
  public static readonly persistenceKey = 'L1ToL2TransactionSynchronizer'

  private readonly topics: string[]
  private readonly topicMap: Map<string, L1ToL2TransactionLogParserContext>

  /**
   * Creates a L1ToL2TransactionSynchronizer that subscribes to blocks, processes all
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
    logContexts: L1ToL2TransactionLogParserContext[],
    listeners: L1ToL2TransactionBatchListener[],
    persistenceKey: string = L1TransactionBatchProcessor.persistenceKey
  ): Promise<L1TransactionBatchProcessor> {
    const processor = new L1TransactionBatchProcessor(
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
    logContexts: L1ToL2TransactionLogParserContext[],
    private readonly listeners: L1ToL2TransactionBatchListener[],
    persistenceKey: string = L1TransactionBatchProcessor.persistenceKey
  ) {
    super(db, persistenceKey)
    this.topicMap = new Map<string, L1ToL2TransactionLogParserContext>(
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

    const l1ToL2TransactionArrays: L1ToL2Transaction[][] = await Promise.all(
      logs.map((l) => this.getTransactionsFromLog(l))
    )
    const transactions: L1ToL2Transaction[] = l1ToL2TransactionArrays.reduce(
      (res, curr) => [...res, ...curr],
      []
    )

    if (!transactions.length) {
      log.debug(`There were no L1toL2Transactions in block ${block.number}.`)
    } else {
      log.debug(
        `Parsed L1ToL2Transactions from block ${block.number}: ${JSON.stringify(
          transactions
        )}`
      )
    }

    this.add(block.number, {
      blockNumber: block.number,
      timestamp: block.timestamp,
      transactions,
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
    transactionBatch: L1ToL2TransactionBatch
  ): Promise<void> {
    try {
      if (!!transactionBatch.transactions.length) {
        this.listeners.map((x) =>
          x.handleL1ToL2TransactionBatch(transactionBatch)
        )
      }
      await this.markProcessed(blockNumber)
    } catch (e) {
      this.logError(
        `Error processing L1ToL2Transactions. Txs: ${JSON.stringify(
          transactionBatch
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
  protected async serializeItem(item: L1ToL2TransactionBatch): Promise<Buffer> {
    return Buffer.from(JSON.stringify(item), 'utf-8')
  }

  /**
   * @inheritDoc
   */
  protected async deserializeItem(
    itemBuffer: Buffer
  ): Promise<L1ToL2TransactionBatch> {
    return JSON.parse(itemBuffer.toString('utf-8'))
  }

  private async getTransactionsFromLog(l: Log): Promise<L1ToL2Transaction[]> {
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

    const parsedTransactions: L1ToL2Transaction[] = []
    for (const topic of matchedTopics) {
      const context = this.topicMap.get(topic)
      if (!addressesAreEqual(l.address, context.contractAddress)) {
        continue
      }
      const transactions = await context.parseL2Transactions(l, transaction)
      parsedTransactions.push(...transactions)
    }

    return parsedTransactions
  }
}
