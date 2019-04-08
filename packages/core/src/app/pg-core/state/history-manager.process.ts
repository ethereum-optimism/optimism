/* Internal Imports */
import { HistoryManager, ChainDB } from '../../../interfaces'
import { Process } from '../../common'
import { PGHistoryManager } from './history-manager'

/**
 * Process that creates a history manager instance.
 */
export class PGHistoryManagerProcoess extends Process<HistoryManager> {
  /**
   * Creates the process.
   * @param chaindb ChainDB used by the history manager.
   */
  constructor(private chaindb: Process<ChainDB>) {
    super()
  }

  /**
   * Creates the instance.
   * Waits for the ChainDB to be ready before
   * resolving the history manager.
   */
  protected async onStart(): Promise<void> {
    await this.chaindb.waitUntilStarted()

    const chaindb = this.chaindb.subject
    this.subject = new PGHistoryManager(chaindb)
  }
}
