/* Internal Imports */
import { Transaction, TransactionProof } from '../../common'

/**
 * StateManager is responsible for updating or
 * querying the local state.
 */
export interface StateManager {
  /**
   * Applies a single transaction to the local state.
   * Will only add the transaction if it's valid given
   * the local known (verified) state and any additional
   * information provided in `TransactionProof`.
   * @param transaction Transaction to apply.
   * @param transactionProof Proof of validity for the transaction.
   */
  applyTransaction(
    transaction: Transaction,
    transactionProof: TransactionProof
  ): Promise<void>

  /**
   * Checks whether a given transaction is valid.
   * Makes use of the local (verified) state as well as
   * any public information (e.g. plasma block headers).
   * @param transaction Transaction to verify.
   * @param transactionProof Proof of validity for the transaction.
   * @returns `true` if the update is valid, `false` otherwise.
   */
  checkTransactionProof(
    transaction: Transaction,
    transactionProof: TransactionProof
  ): Promise<boolean>
}
