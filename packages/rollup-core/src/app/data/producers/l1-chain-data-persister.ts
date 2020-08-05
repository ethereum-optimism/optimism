/* External Imports */
import { DB } from '@eth-optimism/core-db'
import { getLogger, Logger } from '@eth-optimism/core-utils'

import { Block, Provider, TransactionResponse } from 'ethers/providers'
import { Log } from 'ethers/providers/abstract-provider'

/* Internal Imports */
import { L1DataService, LogHandlerContext } from '../../../types'
import { ChainDataProcessor } from './chain-data-processor'

const log: Logger = getLogger('l1-chain-data-persister')

/**
 * This class subscribes to and syncs L1, processing all data of interest and
 * saving it in the DB so that it may be accessed in a more structured way.
 */
export class L1ChainDataPersister extends ChainDataProcessor {
  public static readonly persistenceKey = 'L1ChainDataPersister'
  private readonly topicMap: Map<string, LogHandlerContext>
  private readonly topics: string[]

  /**
   * Creates a L1ChainDataPersister that subscribes to L1 blocks, processes all
   * events, and inserts relevant data into the provided RDB.
   *
   * @param db The DB to use to persist the queue of Block objects.
   * @param dataService The L1 Data Service handling persistence of relevant data.
   * @param l1Provider The provider to use to connect to L1 to subscribe & fetch block / tx / log data.
   * @param logHandlerContexts The collection of LogHandlerContexts that uniquely identify the log events
   *        to be handled and the function that processes their data and inserts them into the RDB.
   * @param persistenceKey The persistence key to use for this instance within the provided DB.
   */
  public static async create(
    db: DB,
    dataService: L1DataService,
    l1Provider: Provider,
    logHandlerContexts: LogHandlerContext[],
    persistenceKey: string = L1ChainDataPersister.persistenceKey
  ): Promise<L1ChainDataPersister> {
    const processor = new L1ChainDataPersister(
      db,
      dataService,
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
    persistenceKey: string
  ) {
    super(db, persistenceKey)
    this.topicMap = new Map<string, LogHandlerContext>(
      this.logHandlerContexts.map((x) => [x.topic, x])
    )

    if (this.topicMap.size !== logHandlerContexts.length) {
      throw Error('There must be exactly one log context for each log topic')
    }
    this.topics = Array.from(this.topicMap.keys())
    log.debug(`topics: ${JSON.stringify(this.topicMap)}`)
  }

  /**
   * @inheritDoc
   */
  protected async handleNextItem(index: number, block: Block): Promise<void> {
    log.debug(
      `handling block ${block.number}. Searching for any relevant logs.`
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

    const relevantLogs = logs
      .filter(
        (x) =>
          x.topics.filter(
            (y) =>
              !!this.topicMap.get(y) &&
              this.topicMap.get(y).contractAddress === x.address
          ).length > 0
      )
      .sort((a, b) => a.logIndex - b.logIndex)

    if (!relevantLogs || !relevantLogs.length) {
      log.debug(
        `No relevant logs found in block ${block.number}. Storing block and moving on.`
      )
      await this.l1DataService.insertL1Block(block, true)
      await this.markProcessed(index)
      return
    }

    log.debug(
      `Handling ${relevantLogs.length} relevant logs from block ${
        block.number
      }: ${JSON.stringify(relevantLogs)}`
    )

    const txs: TransactionResponse[] = await Promise.all(
      relevantLogs.map((l) => this.l1Provider.getTransaction(l.transactionHash))
    )

    await this.l1DataService.insertL1BlockAndTransactions(block, txs, false)

    for (let i = 0; i < relevantLogs.length; i++) {
      const current_log = relevantLogs[i]
      const topics = current_log.topics.filter(
        (x) =>
          !!this.topicMap.get(x) &&
          this.topicMap.get(x).contractAddress === current_log.address
      )
      for (const topic of topics) {
        await this.topicMap
          .get(topic)
          .handleLog(this.l1DataService, current_log, txs[i])
      }
    }

    await this.l1DataService.updateBlockToProcessed(block.hash)

    return this.markProcessed(index)
  }
}
