import { Transaction, TransactionProof } from '../../common'

/**
 * HistoryManager is responsible for storing the full
 * state history and for generating transaction proofs.
 */
export interface HistoryManager {
  /**
   * Adds a set of transactions to the stored history.
   * Should check that it's not storing the same
   * transaction multiple times.
   * @param transactions Transactions to add.
   */
  addTransactions(transactions: Transaction[]): Promise<void>

  /**
   * Creates a proof for a given transaction.
   * Collects all information necessary to prove that the
   * state is actually valid. May assume that the recipient
   * has access to public information (e.g. plasma block headers).
   * @param transaction Transaciton to prove validity of.
   * @returns a proof of validity for that transaction.
   */
  getTransactionProof(transaction: Transaction): Promise<TransactionProof>
}
