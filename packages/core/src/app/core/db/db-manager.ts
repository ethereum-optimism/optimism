import { DBManager, BaseDB, Type } from '../../../interfaces'

/**
 * Basic DBManager implementation that creates instances
 * of a DB type defined at construction time.
 */
export class DefaultDBManager implements DBManager {
  constructor(readonly DefaultDB: Type<BaseDB>) {}

  /**
   * Creates a new database instance.
   * @param args Any arguments to the database.
   * @returns the database instance.
   */
  public create(...args: any[]): BaseDB {
    return new this.DefaultDB(...args)
  }
}
