/* External Imports */
import {
  BaseQueuedPersistedProcessor,
  DB,
  EthereumEvent,
  EthereumListener,
} from '@eth-optimism/core-db'
import { BigNumber, getLogger, Logger } from '@eth-optimism/core-utils'

/* Internal Imports */
import { L1ToL2Transaction, L1ToL2TransactionListener } from '../types'

const log: Logger = getLogger('l1-to-l2-transaction-processor')

export class L1ToL2TransactionProcessor
  extends BaseQueuedPersistedProcessor<L1ToL2Transaction>
  implements EthereumListener<EthereumEvent> {
  public static readonly persistenceKey = 'L1ToL2TransactionProcessor'

  public static async create(
    db: DB,
    l1ToL2EventId: string,
    listeners: L1ToL2TransactionListener[],
    persistenceKey: string = L1ToL2TransactionProcessor.persistenceKey
  ): Promise<L1ToL2TransactionProcessor> {
    const processor = new L1ToL2TransactionProcessor(
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
    private readonly listeners: L1ToL2TransactionListener[],
    persistenceKey: string = L1ToL2TransactionProcessor.persistenceKey
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

    const transaction: L1ToL2Transaction = {
      nonce: event.values['_nonce'].toNumber(),
      sender: event.values['_sender'],
      target: event.values['_target'],
      calldata: event.values['_callData'],
    }

    await this.add(transaction.nonce, transaction)
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
    item: L1ToL2Transaction
  ): Promise<void> {
    try {
      await Promise.all(
        this.listeners.map((x) => x.handleL1ToL2Transaction(item))
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
  protected async serializeItem(item: L1ToL2Transaction): Promise<Buffer> {
    return Buffer.from(JSON.stringify(item), 'utf-8')
  }

  /**
   * @inheritDoc
   */
  protected async deserializeItem(
    itemBuffer: Buffer
  ): Promise<L1ToL2Transaction> {
    return JSON.parse(itemBuffer.toString('utf-8'))
  }
}
