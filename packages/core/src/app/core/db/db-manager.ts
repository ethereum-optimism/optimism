/* External Imports */
import path = require('path')
import { AbstractLevelDOWNConstructor } from 'abstract-leveldown'

/* Internal Imports */
import { BaseDB } from '../../common'
import { DBManager, DB } from '../../../interfaces'

/**
 * Basic DBManager implementation that creates instances
 * of a DB type defined at construction time.
 */
export class SimpleDBManager implements DBManager {
  private cache: Record<string, DB> = {}

  /**
   * Creates the DB manager.
   * @param baseDbPath Base path to create DBs from.
   * @param backend Backend to use when creating new DBs.
   */
  constructor(
    private readonly baseDbPath: string,
    private readonly backend: AbstractLevelDOWNConstructor
  ) {}

  /**
   * Creates a new database instance.
   * @param dbpath Path for the database.
   * @returns the database instance.
   */
  public create(...dbpath: any[]): DB {
    const dbPath = path.join(this.baseDbPath, ...dbpath)
    if (dbPath in this.cache) {
      return this.cache[dbPath]
    }

    const backend = new this.backend(dbPath)
    const db = new BaseDB(backend)
    this.cache[dbPath] = db
    return db
  }
}
