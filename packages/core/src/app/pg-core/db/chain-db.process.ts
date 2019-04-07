import { DBManager, DB, AddressResolver } from '../../../interfaces'
import { Process } from '../../common'
import { ChainDB } from './chain-db'

export class DefaultChainDBProcess extends Process<ChainDB> {
  private db: DB

  constructor(
    private addressResolver: Process<AddressResolver>,
    private dbManager: Process<DBManager>
  ) {
    super()
  }

  protected async onStart(): Promise<void> {
    await this.addressResolver.waitUntilStarted()
    await this.dbManager.waitUntilStarted()

    const address = this.addressResolver.subject.address
    this.db = this.dbManager.subject.create(address)
    await this.db.open()
    this.subject = new ChainDB(this.db)
  }
}
