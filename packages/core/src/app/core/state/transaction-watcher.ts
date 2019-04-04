import { BaseRunnable } from '../app-thing'

/**
 * Responsible for watching for new transactions.
 */
export class DefaultTransactionWatcher extends BaseRunnable {
  constructor(private nodeRpcClient) {
    super()
  }

  public async onStart(): Promise<void> {
    // Loop and repeatedly check for new transactions that impact me.
    // If we see a transaction that impacts me, pull the proof and then emit an event.
  }
}
