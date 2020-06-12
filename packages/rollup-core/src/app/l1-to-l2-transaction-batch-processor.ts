/* External Imports */
import {
  BaseQueuedPersistedProcessor,
  DB,
  EthereumEvent,
  EthereumListener,
} from '@eth-optimism/core-db'
import { getLogger, logError, Logger } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  L1ToL2Transaction,
  L1ToL2TransactionBatch,
  L1ToL2TransactionBatchListener,
} from '../types'
import { Provider, TransactionResponse } from 'ethers/providers'

const log: Logger = getLogger('l1-to-l2-transition-batch-processor')

export class L1ToL2TransactionBatchProcessor
  extends BaseQueuedPersistedProcessor<L1ToL2TransactionBatch>
  implements EthereumListener<EthereumEvent> {
  public static readonly persistenceKey = 'L1ToL2TransitionBatchProcessor'

  private readonly provider: Provider

  public static async create(
    db: DB,
    l1ToL2EventId: string,
    listeners: L1ToL2TransactionBatchListener[],
    persistenceKey: string = L1ToL2TransactionBatchProcessor.persistenceKey
  ): Promise<L1ToL2TransactionBatchProcessor> {
    const processor = new L1ToL2TransactionBatchProcessor(
      db,
      l1ToL2EventId,
      listeners,
      persistenceKey
    )
    await processor.init()
    return processor
  }

  private constructor(
    db: DB,
    private readonly l1ToL2EventId: string,
    private readonly listeners: L1ToL2TransactionBatchListener[],
    persistenceKey: string = L1ToL2TransactionBatchProcessor.persistenceKey
  ) {
    super(db, persistenceKey)
  }

  /**
   * @inheritDoc
   */
  public async handle(event: EthereumEvent): Promise<void> {
    if (event.eventID !== this.l1ToL2EventId || !event.values) {
      log.debug(
        `Received event of wrong ID or with incorrect values. Ignoring event: [${JSON.stringify(
          event
        )}]`
      )
      return
    }

    const calldata = await this.fetchCalldata(event.transactionHash)

    let transactions: L1ToL2Transaction[] = []
    try {
      transactions = await this.parseTransactions(calldata)
    } catch (e) {
      // TODO: What do we do here?
      logError(
        log,
        `Error parsing calldata for event ${JSON.stringify(
          event
        )}. Assuming this tx batch was malicious / invalid. Moving on.`,
        e
      )
    }

    const transactionBatch: L1ToL2TransactionBatch = {
      nonce: event.values['_nonce'].toNumber(),
      timestamp: event.values['_timestamp'].toNumber(),
      transactions,
      calldata,
    }

    this.add(transactionBatch.nonce, transactionBatch)
  }

  /**
   * @inheritDoc
   */
  public async onSyncCompleted(syncIdentifier?: string): Promise<void> {
    // no-op
  }

  /**
   * @inheritDoc
   */
  protected async handleNextItem(
    index: number,
    item: L1ToL2TransactionBatch
  ): Promise<void> {
    try {
      await Promise.all(
        this.listeners.map((x) => x.handleTransactionBatch(item))
      )
      await this.markProcessed(index)
    } catch (e) {
      this.logError(
        `Error processing L1ToL2Transaction in at least one handler. Tx: ${JSON.stringify(
          item
        )}`,
        e
      )
      // All errors should be caught in the listeners, so this is fatal.
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

  private async fetchCalldata(txHash: string): Promise<string> {
    let tx: TransactionResponse
    try {
      tx = await this.provider.getTransaction(txHash)
    } catch (e) {
      logError(
        log,
        `Error fetching tx hash ${txHash}. This should not ever happen.`,
        e
      )
      process.exit(1)
    }

    return tx.data
  }

  // TODO: This when a format is solidified
  private async parseTransactions(
    calldata: string
  ): Promise<L1ToL2Transaction[]> {
    return []
  }
}
