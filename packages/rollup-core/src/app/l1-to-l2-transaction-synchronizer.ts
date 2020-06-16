/* External Imports */
import {
  BaseQueuedPersistedProcessor,
  DB,
  EthereumListener,
} from '@eth-optimism/core-db'
import {
  getLogger,
  Logger,
  numberToHexString,
  remove0x,
} from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  L1ToL2Transaction,
  TimestampedL1ToL2Transactions,
  L1ToL2TransactionLogParserContext,
} from '../types'
import {
  Block,
  JsonRpcProvider,
  Provider,
  TransactionResponse,
} from 'ethers/providers'
import { Log } from 'ethers/providers/abstract-provider'
import { Wallet } from 'ethers'
import { addressesAreEqual } from './utils'

const log: Logger = getLogger('l1-to-l2-transition-synchronizer')

// params: [timestampHex, transactionsArrayJSON, signedTransactionsArrayJSON]
const sendL1ToL2TransactionsMethod: string = 'optimism_sendL1ToL2Transactions'

export class L1ToL2TransactionSynchronizer
  extends BaseQueuedPersistedProcessor<TimestampedL1ToL2Transactions>
  implements EthereumListener<Block> {
  public static readonly persistenceKey = 'L1ToL2TransactionSynchronizer'

  private readonly topics: string[]
  private readonly topicMap: Map<string, L1ToL2TransactionLogParserContext>
  private readonly l2Provider: JsonRpcProvider

  /**
   * Creates a L1ToL2TransactionSynchronizer that subscribes to blocks, processes all
   * L1ToL2Transaction events, parses L1ToL2Transactions and submits them to L2.
   *
   * @param db The DB to use to persist the queue of L1ToL2Transaction[] objects.
   * @param l1Provider The provider to use to connect to L1 to subscribe & fetch block / tx / log data.
   * @param l2Wallet The L2 wallet to use to submit transactions ** ASSUMED TO BE CONNECTED TO THE L2 JSON RPC PROVIDER **
   * @param logContexts The collection of L1ToL2TransactionLogParserContext that uniquely identify the log event and
   *        provide the ability to create L2 transactions from the L1 transaction that emitted it.
   * @param persistenceKey The persistence key to use for this instance within the provided DB.
   */
  public static async create(
    db: DB,
    l1Provider: Provider,
    l2Wallet: Wallet,
    logContexts: L1ToL2TransactionLogParserContext[],
    persistenceKey: string = L1ToL2TransactionSynchronizer.persistenceKey
  ): Promise<L1ToL2TransactionSynchronizer> {
    const processor = new L1ToL2TransactionSynchronizer(
      db,
      l1Provider,
      l2Wallet,
      logContexts,
      persistenceKey
    )
    await processor.init()
    return processor
  }

  private constructor(
    db: DB,
    private readonly l1Provider: Provider,
    private readonly l2Wallet: Wallet,
    logContexts: L1ToL2TransactionLogParserContext[],
    persistenceKey: string = L1ToL2TransactionSynchronizer.persistenceKey
  ) {
    super(db, persistenceKey)
    this.topicMap = new Map<string, L1ToL2TransactionLogParserContext>(
      logContexts.map((x) => [x.topic, x])
    )
    this.topics = Array.from(this.topicMap.keys())
    this.l2Provider = l2Wallet.provider as JsonRpcProvider
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
    timestampedTransactions: TimestampedL1ToL2Transactions
  ): Promise<void> {
    try {
      if (
        !timestampedTransactions.transactions ||
        !timestampedTransactions.transactions.length
      ) {
        log.debug(`Moving past empty block ${blockNumber}.`)
        await this.markProcessed(blockNumber)
        return
      }

      const timestamp: string = numberToHexString(
        timestampedTransactions.timestamp
      )
      const txs = JSON.stringify(
        timestampedTransactions.transactions.map((x) => {
          return {
            nonce: x.nonce > 0 ? numberToHexString(x.nonce) : '',
            sender: x.sender,
            calldata: x.calldata,
          }
        })
      )
      const signedTxsArray: string = await this.l2Wallet.signMessage(txs)
      await this.l2Provider.send(sendL1ToL2TransactionsMethod, [
        timestamp,
        txs,
        signedTxsArray,
      ])

      await this.markProcessed(blockNumber)
    } catch (e) {
      this.logError(
        `Error processing L1ToL2Transactions. Txs: ${JSON.stringify(
          timestampedTransactions
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
  protected async serializeItem(
    item: TimestampedL1ToL2Transactions
  ): Promise<Buffer> {
    return Buffer.from(JSON.stringify(item), 'utf-8')
  }

  /**
   * @inheritDoc
   */
  protected async deserializeItem(
    itemBuffer: Buffer
  ): Promise<TimestampedL1ToL2Transactions> {
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

    const parsedTransactions: L1ToL2Transaction[] = []
    for (const topic of matchedTopics) {
      const context = this.topicMap.get(topic)
      if (!addressesAreEqual(l.address, context.contractAddress)) {
        continue
      }
      const transactions = await context.parseL2Transactions(transaction)
      parsedTransactions.push(...transactions)
    }

    return parsedTransactions
  }
}
