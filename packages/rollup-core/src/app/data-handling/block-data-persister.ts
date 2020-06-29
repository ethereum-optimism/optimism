/* External Imports */
import {
  BaseQueuedPersistedProcessor,
  DB,
  EthereumListener,
} from '@eth-optimism/core-db'
import { BigNumber, getLogger, Logger } from '@eth-optimism/core-utils'

import { Block, Provider, TransactionResponse } from 'ethers/providers'
import { L1DataService, LogHandlerContext } from '../../types'
import { Log } from 'ethers/providers/abstract-provider'

/* Internal Imports */

const log: Logger = getLogger('block-data-persister')

export class BlockDataPersister extends BaseQueuedPersistedProcessor<Block>
  implements EthereumListener<Block> {
  public static readonly persistenceKey
  private readonly topicMap: Map<string, LogHandlerContext>
  private readonly topics: string[]

  /**
   * Creates a BlockDataPersister that subscribes to blocks, processes all
   * events, and inserts relevant data into the provided RDB.
   *
   * @param db The DB to use to persist the queue of Block objects.
   * @param l1DataService The L1 Data Service handling persistence of relevant data.
   * @param l1Provider The provider to use to connect to L1 to subscribe & fetch block / tx / log data.
   * @param logHandlerContexts The collection of LogHandlerContexts that uniquely identify the log events
   *        to be handled and the function that processes their data and inserts them into the RDB.
   * @param persistenceKey The persistence key to use for this instance within the provided DB.
   */
  public static async create(
    db: DB,
    l1DataService: L1DataService,
    l1Provider: Provider,
    logHandlerContexts: LogHandlerContext[],
    persistenceKey: string = BlockDataPersister.persistenceKey
  ): Promise<BlockDataPersister> {
    const processor = new BlockDataPersister(
      db,
      l1DataService,
      l1Provider,
      logHandlerContexts,
      persistenceKey
    )
    await processor.init()
    return processor
  }

  private constructor(
    db: DB,
    private readonly l1DataService: L1DataService,
    private readonly l1Provider: Provider,
    private readonly logHandlerContexts: LogHandlerContext[],
    persistenceKey: string = BlockDataPersister.persistenceKey
  ) {
    super(db, persistenceKey)
    this.topicMap = new Map<string, LogHandlerContext>(
      logHandlerContexts.map((x) => [x.topic, x])
    )
    if (this.topicMap.size !== logHandlerContexts.length) {
      throw Error('There may only be one log context for each log topic')
    }
    this.topics = Array.from(this.topicMap.keys())
  }

  /**
   * @inheritDoc
   */
  public async handle(block: Block): Promise<void> {
    log.debug(`Received block ${block.number}.`)

    // purposefully not awaited
    this.add(block.number, block)
  }

  /**
   * @inheritDoc
   */
  protected async handleNextItem(index: number, block: Block): Promise<void> {
    log.debug(
      `handling block ${block.number}. Searching for any relevant logs.`
    )

    let logs: Log[] = await this.l1Provider.getLogs({
      blockHash: block.hash,
      topics: this.topics,
    })
    log.debug(
      `Got ${logs.length} logs from block ${block.number}: ${JSON.stringify(
        logs
      )}`
    )

    logs = logs
      .filter((x) =>
        x.topics.filter(
          (y) =>
            !!this.topicMap.get(y) &&
            this.topicMap.get(y).contractAddress === x.address
        )
      )
      .sort((a, b) => a.logIndex - b.logIndex)

    const txs: TransactionResponse[] = await Promise.all(
      logs.map((l) => this.l1Provider.getTransaction(l.transactionHash))
    )

    await this.l1DataService.insertBlockAndTransactions(block, txs, false)

    const handlerPromises: Array<Promise<any>> = []
    for (let i = 0; i < logs.length; i++) {
      const current_log = logs[i]
      const topics = current_log.topics.filter(
        (x) =>
          !!this.topicMap.get(x) &&
          this.topicMap.get(x).contractAddress === current_log.address
      )
      for (const topic of topics) {
        handlerPromises.push(
          this.topicMap.get(topic).handleLog(current_log, txs[i])
        )
      }
    }

    await Promise.all(handlerPromises)

    await this.l1DataService.updateBlockToProcessed(block.hash)
  }

  /**
   * @inheritDoc
   */
  public async onSyncCompleted(syncIdentifier?: string): Promise<void> {
    return undefined
  }

  /**
   * @inheritDoc
   */
  protected async deserializeItem(itemBuffer: Buffer): Promise<Block> {
    return JSON.parse(itemBuffer.toString('utf-8'), (key, val) => {
      if (key === 'gasLimit' || key === 'gasUsed') {
        return !!val ? new BigNumber(val) : undefined
      }
      return val
    })
  }

  /**
   * @inheritDoc
   */
  protected async serializeItem(item: Block): Promise<Buffer> {
    return Buffer.from(
      JSON.stringify(item, (key, val) => {
        if (key === 'gasLimit' || key === 'gasUsed') {
          try {
            return val.toString('hex')
          } catch (e) {
            // need to use null because undefined will omit the value.
            return null
          }
        }
        return val
      }),
      'hex'
    )
  }
}
