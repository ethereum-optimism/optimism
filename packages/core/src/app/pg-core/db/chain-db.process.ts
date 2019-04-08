/* Internal Imports */
import { DBManager, DB, AddressResolver, ChainDB } from '../../../interfaces'
import { Process } from '../../common'
import { PGChainDB } from './chain-db'

/**
 * Process that creates a ChainDB instance.
 */
export class PGChainDBProcess extends Process<ChainDB> {
  private db: DB

  /**
   * Creates the process.
   * @param addressResolver Module that resolves the plasma chain address.
   * @param dbManager DB manager used to open the underlying DB.
   */
  constructor(
    private addressResolver: Process<AddressResolver>,
    private dbManager: Process<DBManager>
  ) {
    super()
  }

  /**
   * Creates the ChainDB instance.
   * Waits for the address resolver and DB manager
   * to be ready before opening the appropriate DB.
   */
  protected async onStart(): Promise<void> {
    await this.addressResolver.waitUntilStarted()
    await this.dbManager.waitUntilStarted()

    const address = this.addressResolver.subject.address
    this.db = this.dbManager.subject.create(address)
    await this.db.open()
    this.subject = new PGChainDB(this.db)
  }
}
