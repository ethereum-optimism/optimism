import { StateManager, ChainDB } from '../../../interfaces'
import { Process } from '../../common'
import { PGStateManager } from './state-manager'

export class PGStateManagerProcess extends Process<StateManager> {
  constructor(private chaindb: Process<ChainDB>) {
    super()
  }

  protected async onStart(): Promise<void> {
    await this.chaindb.waitUntilStarted()

    const chaindb = this.chaindb.subject
    this.subject = new PGStateManager(chaindb)
  }
}
