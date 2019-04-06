import {
  MessageBus,
  StateManager,
  Transaction,
  TransactionProof,
} from '../../../interfaces'

/**
 * Simples transaction receiver that listens for incoming
 * transaction events and inserts them into the database.
 */
export class DefaultTransactionReceiver {
  constructor(
    private messageBus: MessageBus,
    private stateManager: StateManager
  ) {}

  public async onStart(): Promise<void> {
    this.messageBus.on(
      'transaction:new',
      (transaction: Transaction, transactionProof: TransactionProof) => {
        this.stateManager.applyTransaction(transaction, transactionProof)
      }
    )
  }
}
