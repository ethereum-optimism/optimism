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
   * @param earliestBlock The earliest block to sync.
   * @param persistenceKey The persistence key to use for this instance within the provided DB.
   */
  public static async create(
    db: DB,
    dataService: L1DataService,
    l1Provider: Provider,
    logHandlerContexts: LogHandlerContext[],
    earliestBlock: number = 0,
    persistenceKey: string = L1ChainDataPersister.persistenceKey
  ): Promise<L1ChainDataPersister> {
    const processor = new L1ChainDataPersister(
      db,
      dataService,
      l1Provider,
      logHandlerContexts,
      earliestBlock,
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
    private earliestBlock: number,
    persistenceKey: string
  ) {
    super(db, persistenceKey, earliestBlock)
    this.topicMap = new Map<string, LogHandlerContext>(
      this.logHandlerContexts.map((x) => [x.topic, x])
    )

    if (this.topicMap.size !== this.logHandlerContexts.length) {
      throw Error('There must be exactly one log context for each log topic')
    }
    this.topics = Array.from(this.topicMap.keys())
    log.debug(`topics: ${JSON.stringify(this.topics)}`)
  }

  /**
   * @inheritDoc
   */
  protected async handleNextItem(index: number, block: Block): Promise<void> {
    log.debug(
      `handling block ${block.number}. Searching for any relevant logs.`
    )

    let relevantLogs: Log[]
    let txs: TransactionResponse[]

    try {
      const logs: Log[] = await this.getLogsForBlock(block.hash)

      log.debug(
        `Got ${logs.length} logs from block ${block.number}: ${JSON.stringify(
          logs
        )}`
      )

      relevantLogs = logs
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

      txs = await Promise.all(
        relevantLogs.map((l) =>
          this.l1Provider.getTransaction(l.transactionHash)
        )
      )
    } catch (e) {
      this.logError(`Error parsing block ${block.number}`, e)
      throw e
    }

    try {
      log.debug(
        `Inserting block ${block.number} and ${txs.length} transactions.`
      )
      await this.l1DataService.insertL1BlockAndTransactions(block, txs, false)

      log.debug(
        `Looping through ${relevantLogs.length} logs from block ${block.number} to insert rollup transactions & state roots`
      )
      for (const [i, currentLog] of relevantLogs.entries()) {
        const topics = currentLog.topics.filter(
          (x) =>
            !!this.topicMap.get(x) &&
            this.topicMap.get(x).contractAddress === currentLog.address
        )
        for (const topic of topics) {
          await this.topicMap
            .get(topic)
            .handleLog(this.l1DataService, currentLog, txs[i])
        }
      }

      await this.l1DataService.updateBlockToProcessed(block.hash)
    } catch (e) {
      this.logError(
        `Error inserting block & tx data for block ${block.number}`,
        e
      )
      throw e
    }

    return this.markProcessed(index)
  }

  /**
   * Gets the logs for a given block that match our topics, taking into account the fact that
   * we have to do a separate fetch for each topic.
   *
   * @param blockHash The block hash for the block in which we're searching for logs.
   * @returns The combined array of Logs that we care about based on our topics.
   */
  private async getLogsForBlock(blockHash: string): Promise<Log[]> {
    if (!this.topics.length) {
      return []
    }

    const logsArrays: Log[][] = await Promise.all(
      this.topics.map((topic) =>
        this.l1Provider.getLogs({
          blockHash,
          topics: [topic],
        })
      )
    )

    const flattened: Log[] = [].concat(...logsArrays)
    log.debug(`Logs Arrays: ${JSON.stringify(flattened)}`)

    return flattened
  }
}
