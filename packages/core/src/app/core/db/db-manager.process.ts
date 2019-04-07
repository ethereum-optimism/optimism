import { ConfigManager, DBManager } from '../../../interfaces'
import { Process } from '../../common'
import { DefaultDBManager } from './db-manager'

export class DefaultDBManagerProxy extends Process<DBManager> {
  constructor(private config: Process<ConfigManager>) {
    super()
  }

  protected async onStart(): Promise<void> {
    await this.config.waitUntilStarted()
    const baseDbPath = this.config.subject.get('BASE_DB_PATH')
    const dbBackend = this.config.subject.get('DB_BACKEND')
    this.subject = new DefaultDBManager(baseDbPath, dbBackend)
  }
}
