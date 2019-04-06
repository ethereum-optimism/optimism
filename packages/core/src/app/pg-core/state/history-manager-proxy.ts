import { MessageBus } from '../../../interfaces'
import { ProxyProcess } from '../../common'
import { PGHistoryManager } from './history-manager'
import { ChainDB } from '../db/chain-db'

export class PGHistoryManagerProxy extends ProxyProcess<PGHistoryManager> {
  constructor(private messageBus: MessageBus, private chaindb: ChainDB) {
    super()
  }

  protected async onStart(): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      this.messageBus.on('CHAIN_DB_READY', () => {
        this.instance = new PGHistoryManager(this.chaindb)
        resolve()
      })
    })
  }
}
