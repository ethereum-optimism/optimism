import { DBManager, DB, AddressResolver, ChainDB } from '../../../interfaces'
import { Process } from '../../common'
import { PGChainDB } from './chain-db'

export class PGChainDBProcess extends Process<ChainDB> {
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
    this.subject = new PGChainDB(this.db)
  }
}
