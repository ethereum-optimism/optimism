import { DBManager, DB, KeyManager } from '../../../interfaces'
import { Process } from '../../common'
import { DefaultKeyManager } from './key-manager'

export class DefaultKeyManagerProcess extends Process<KeyManager> {
  private db: DB

  constructor(private dbManager: Process<DBManager>) {
    super()
  }

  protected async onStart(): Promise<void> {
    await this.dbManager.waitUntilStarted()

    this.db = this.dbManager.subject.create('keys')
    await this.db.open()
    this.subject = new DefaultKeyManager(this.db)
  }

  protected async onStop() {
    await this.db.close()
  }
}
