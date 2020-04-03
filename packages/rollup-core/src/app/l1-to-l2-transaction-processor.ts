import {BaseQueuedPersistedProcessor, DB, EthereumEvent, EthereumListener} from '@eth-optimism/core-db'
import {L1ToL2Transaction, L1ToL2TransactionListener} from '../types'
import {L1ToL2TransactionEventId} from './constants'

export class L1ToL2TransactionProcessor
  extends BaseQueuedPersistedProcessor<L1ToL2Transaction>
  implements EthereumListener<EthereumEvent> {

  public static readonly persistenceKey = 'L1ToL2TransactionProcessor'

  public static async create(db: DB, listeners: L1ToL2TransactionListener[], persistenceKey: string = L1ToL2TransactionProcessor.persistenceKey): Promise<L1ToL2TransactionProcessor> {
    const processor = new L1ToL2TransactionProcessor(db, listeners, persistenceKey)
    await processor.init()
    return processor
  }

  private constructor(db: DB, private readonly listeners: L1ToL2TransactionListener[],  persistenceKey: string = L1ToL2TransactionProcessor.persistenceKey) {
    super(db, persistenceKey);
  }

  /**
   * @inheritDoc
   */
  public async handle(event: EthereumEvent): Promise<void> {
    if (event.eventID !== L1ToL2TransactionEventId) {
      return
    }

    const transaction: L1ToL2Transaction = {
      nonce: event.values['_nonce'],
      sender: event.values['_sender'],
      target: event.values['_target'],
      callData: event.values['_callData']
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
  protected async handleNextItem(index: number, item: L1ToL2Transaction): Promise<void> {
    try {
      await Promise.all(this.listeners.map(x => x.handleL1ToL2Transaction(item)))
      await this.markProcessed(index)
    } catch (e) {
      this.logError(`Error processing L1ToL2Transaction in at least one handler. Tx: ${JSON.stringify(item)}`, e)
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
  protected async deserializeItem(itemBuffer: Buffer): Promise<L1ToL2Transaction> {
    return JSON.parse(itemBuffer.toString('utf-8'))
  }
}