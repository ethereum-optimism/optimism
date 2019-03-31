import { DBManager, BaseDB, Type } from '../../../interfaces'

export class DefaultDBManager implements DBManager {
  constructor(private DB: Type<BaseDB>) {}

  /**
   * Creates a new database instance.
   * @param args Any arguments to the database.
   * @returns the database instance.
   */
  public create(...args: any[]): BaseDB {
    return new this.DB(...args)
  }
}
