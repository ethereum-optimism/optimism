import { BaseRunnable } from '../../common'
import { Service } from '@nestd/core';

/**
 * Responsible for watching for new transactions.
 */
@Service()
export class DefaultTransactionWatcher extends BaseRunnable {
  public async onStart(): Promise<void> {
    // Loop and repeatedly check for new transactions that impact me.
    // If we see a transaction that impacts me, pull the proof and then emit an event.
  }
}
