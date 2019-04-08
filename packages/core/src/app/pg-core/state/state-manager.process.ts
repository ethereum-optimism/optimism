/* Internal Imports */
import { StateManager, ChainDB } from '../../../interfaces'
import { Process } from '../../common'
import { PGStateManager } from './state-manager'

/**
 * Process that creates a state manager instance.
 */
export class PGStateManagerProcess extends Process<StateManager> {
  /**
   * Creates the process.
   * @param chaindb ChainDB used by the state manager.
   */
  constructor(private chaindb: Process<ChainDB>) {
    super()
  }

  /**
   * Creates the instance.
   * Waits for the ChainDB to be ready before
   * resolving the state manager.
   */
  protected async onStart(): Promise<void> {
    await this.chaindb.waitUntilStarted()

    const chaindb = this.chaindb.subject
    this.subject = new PGStateManager(chaindb)
  }
}
