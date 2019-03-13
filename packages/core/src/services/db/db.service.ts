/* External Imports */
import { Service } from '@nestd/core'
import { Type } from '@nestd/core/src/interfaces'

/* Services */
import { ConfigService } from '../config.service'

/* Internal Imports */
import { BaseDBProvider, DBOptions } from './backends/base-db.provider'
import { CONFIG } from '../../constants'

/**
 * Service that handles connecting to the various
 * databases that compose the app.
 */
@Service()
export class DBService {
  public dbs: { [key: string]: BaseDBProvider } = {}

  constructor(private readonly config: ConfigService) {}

  /**
   * Opens a new database with the given name.
   * @param options Any additional options to the provider.
   * @param provider The database provider.
   */
  public async open(
    options: DBOptions,
    provider?: Type<BaseDBProvider>
  ): Promise<BaseDBProvider> {
    // Return the database if it already exists.
    const name = options.name + (options.id ? `.${options.id}` : '')
    if (name in this.dbs) {
      return this.dbs[name]
    }

    // Otherwise create a new database instance.
    const DbProvider = provider || this.dbProvider()
    const db = new DbProvider({ ...options })
    await db.start()
    this.dbs[name] = db

    // Finally, return the database.
    return db
  }

  /**
   * @returns the current default database provider.
   */
  private dbProvider(): Type<BaseDBProvider> {
    return this.config.get(CONFIG.DB_PROVIDER)
  }
}
