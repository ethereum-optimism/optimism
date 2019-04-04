import {
  MessageBus,
  StateManager,
  Transaction,
  TransactionProof,
} from '../../../interfaces'
import { BaseRunnable } from '../../common'
import { DefaultMessageBus } from '../../common/app/message-bus';
import { Service } from '@nestd/core';
import { PGStateManagerHost } from './state-manager-host';

/**
 * Simples transaction receiver that listens for incoming
 * transaction events and inserts them into the database.
 */
@Service()
export class DefaultTransactionReceiver extends BaseRunnable {
  constructor(
    private messageBus: DefaultMessageBus,
    private stateManager: PGStateManagerHost
  ) {
    super()
  }

  public async onStart(): Promise<void> {
    this.messageBus.on(
      'transaction:new',
      (transaction: Transaction, transactionProof: TransactionProof) => {
        this.stateManager.stateManager.applyTransaction(transaction, transactionProof)
      }
    )
  }
}
