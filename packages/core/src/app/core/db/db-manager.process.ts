import { ConfigManager, DBManager } from '../../../interfaces'
import { Process } from '../../common'
import { CORE_CONFIG_KEYS } from '../constants'
import { SimpleDBManager } from './db-manager'

/**
 * Process that initializes a DB manager instance.
 */
export class SimpleDBManagerProcess extends Process<DBManager> {
  /**
   * Creates the process.
   * @param config Config process used to load config values.
   */
  constructor(private config: Process<ConfigManager>) {
    super()
  }

  /**
   * Creates the DB manager instance.
   * Waits for config to be ready and then loads
   * necessary config values.
   */
  protected async onStart(): Promise<void> {
    await this.config.waitUntilStarted()

    const baseDbPath = this.config.subject.get(CORE_CONFIG_KEYS.BASE_DB_PATH)
    const dbBackend = this.config.subject.get(CORE_CONFIG_KEYS.DB_BACKEND)
    this.subject = new SimpleDBManager(baseDbPath, dbBackend)
  }
}
