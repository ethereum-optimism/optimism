import { MessageBus, DBManager, DB } from '../../../interfaces'
import { ProxyProcess } from '../../common'
import { ChainDB } from './chain-db'

export class DefaultChainDBProcess extends ProxyProcess<ChainDB> {
  private db: DB

  constructor(private messageBus: ProxyProcess<MessageBus>, private dbManager: ProxyProcess<DBManager>) {
    super()
  }

  protected async onStart(): Promise<void> {
    await this.messageBus.waitUntilStarted()
    await this.dbManager.waitUntilStarted()

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
