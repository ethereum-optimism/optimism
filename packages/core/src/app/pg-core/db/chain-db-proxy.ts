import { MessageBus, DBManager, DB } from '../../../interfaces'
import { ProxyProcess } from '../../common'
import { ChainDB } from './chain-db'

export class ChainDBProxy extends ProxyProcess<ChainDB> {
  private db: DB

  constructor(private messageBus: MessageBus, private dbManager: DBManager) {
    super()
  }

  protected async onStart(): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      this.messageBus.on('ADDRESS_FOUND', async (address: string) => {
        this.db = this.dbManager.create(address)
        await this.db.open()
        this.instance = new ChainDB(this.db)
        this.messageBus.emit('CHAIN_DB_READY')
        resolve()
      })
    })
  }
}
