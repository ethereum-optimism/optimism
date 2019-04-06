import { MessageBus } from '../../../interfaces'
import { ProxyProcess } from '../../common'
import { PGStateManager } from './state-manager'
import { ChainDB } from '../db/chain-db'

export class PGStateManagerProxy extends ProxyProcess<PGStateManager> {
  constructor(private messageBus: MessageBus, private chaindb: ChainDB) {
    super()
  }

  protected async onStart(): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      this.messageBus.on('CHAIN_DB_READY', () => {
        this.instance = new PGStateManager(this.chaindb)
      })
    })
  }
}
