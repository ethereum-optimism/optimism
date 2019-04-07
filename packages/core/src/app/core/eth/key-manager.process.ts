import { DBManager, DB, KeyManager } from '../../../interfaces'
import { Process } from '../../common'
import { SimpleKeyManager } from './key-manager'

/**
 * Process that creates a KeyManager instance.
 */
export class SimpleKeyManagerProcess extends Process<KeyManager> {
  private db: DB

  /**
   * Creates the process.
   * @param dbManager DB manager used to create the key DB.
   */
  constructor(private dbManager: Process<DBManager>) {
    super()
  }

  /**
   * Creates the instance.
   * Waits for the DB manager to be available
   * before creating and opening the key DB.
   */
  protected async onStart(): Promise<void> {
    await this.dbManager.waitUntilStarted()

    this.db = this.dbManager.subject.create('keys')
    await this.db.open()
    this.subject = new SimpleKeyManager(this.db)
  }

  /**
   * Closes the key DB before shutdown.
   */
  protected async onStop() {
    await this.db.close()
  }
}
