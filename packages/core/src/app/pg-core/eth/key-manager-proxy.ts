import { ProxyProcess } from '../../common'
import { DBManager, DB } from '../../../interfaces'
import { DefaultKeyManager } from '../../core'

export class DefaultKeyManagerProxy extends ProxyProcess<DefaultKeyManager> {
  private db: DB

  constructor(private dbManager: DBManager) {
    super()
  }

  protected async onStart(): Promise<void> {
    this.db = this.dbManager.create('keys')
    await this.db.open()
    this.instance = new DefaultKeyManager(this.db)
  }

  protected async onStop(): Promise<void> {
    await this.db.close()
  }
}
