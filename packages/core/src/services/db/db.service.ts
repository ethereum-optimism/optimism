/* External Imports */
import { Service } from '@nestd/core'

/* Internal Imports */
import { BaseDBProvider } from './backends/base-provider'
import { EphemDBProvider } from './backends/ephem-provider'

/**
 * Service that handles connecting to the various
 * databases that compose the app.
 */
@Service()
export class DBService {
  public dbs: { [key: string]: BaseDBProvider } = {}

  /**
   * Opens a new database with the given name.
   * @param name Name of the new database.
   * @param options Any additional options to the provider.
   * @param provider The database provider.
   */
  public async open(
    name: string,
    options: {} = {},
    provider = this.options.dbProvider
  ): Promise<void> {
    if (name in this) {
      return
    }

    const db = new provider({ ...{ name }, ...options })
    await db.start()
    this.dbs[name] = db
  }
}
