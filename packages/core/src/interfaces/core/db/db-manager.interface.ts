import { DB } from '../../common'

/**
 * DBManager manages database instances.
 */
export interface DBManager {
  /**
   * Creates a new database instance with some given args.
   * @param args to initialize the database with.
   * @returns the database instance.
   */
  create(...args: any[]): DB
}
