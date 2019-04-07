import { ConfigManager, DBManager } from '../../../interfaces'
import { Process } from '../../common'
import { CORE_CONFIG_KEYS } from '../constants'
import { DefaultDBManager } from './db-manager'

export class DefaultDBManagerProxy extends Process<DBManager> {
  constructor(private config: Process<ConfigManager>) {
    super()
  }

  protected async onStart(): Promise<void> {
    await this.config.waitUntilStarted()
    const baseDbPath = this.config.subject.get(CORE_CONFIG_KEYS.BASE_DB_PATH)
    const dbBackend = this.config.subject.get(CORE_CONFIG_KEYS.DB_BACKEND)
    this.subject = new DefaultDBManager(baseDbPath, dbBackend)
  }
}
