/* Internal Imports */
import { TransactionAndRoot } from '../types'

export interface L2DataService {
  /**
   * Inserts the provided L2 transaction into the associated RDB.
   *
   * @param transaction The transaction to insert.
   * @throws An error if there is a DB error.
   */
  insertL2Transaction(transaction: TransactionAndRoot): Promise<void>
}
