import {
  MessageBus,
  StateManager,
  Transaction,
  TransactionProof,
} from '../../../interfaces'
import { BaseRunnable } from '../app-thing'

/**
 * Simples transaction receiver that listens for incoming
 * transaction events and inserts them into the database.
 */
export class DefaultTransactionReceiver extends BaseRunnable {
  constructor(
    private messageBus: MessageBus,
    private stateManager: StateManager
  ) {
    super()
  }

  public async onStart(): Promise<void> {
    this.messageBus.on(
      'transaction:new',
      (transaction: Transaction, transactionProof: TransactionProof) => {
        this.stateManager.applyTransaction(transaction, transactionProof)
      }
    )
  }
}
