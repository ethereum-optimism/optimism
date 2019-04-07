import { StateManager } from '../../../interfaces'
import { Process } from '../../common'
import { PGStateManager } from './state-manager'
import { ChainDB } from '../db/chain-db'

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
