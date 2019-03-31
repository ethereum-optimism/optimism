import {
  HistoryManager,
  Transaction,
  TransactionProof,
} from '../../../interfaces'

/**
 * HistoryManager implementation for PG's Plasma Cashflow variant.
 */
export class PGHistoryManager implements HistoryManager {
  /**
   * Adds a set of transactions to the local state.
   * Does nothing if the transaction is already in the state.
   * @param transactions Set of transactions to add.
   */
  public async addTransactions(transactions: Transaction[]): Promise<void> {}

  /**
   * Generates a transaction proof for a given transaction.
   * Assumes that the recipient has access to public
   * information (e.g. plasma blocks).
   * @param transaction Transaction to generate a proof for.
   * @returns the transaction proof.
   */
  public async getTransactionProof(
    transaction: Transaction
  ): Promise<TransactionProof> {}
}
