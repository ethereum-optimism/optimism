import {
  StateManager,
  Transaction,
  TransactionProof,
} from '../../../interfaces'

/**
 * StateManager implementation for PG's Plasma Cashflow variant.
 */
export class PGStateManager implements StateManager {
  /**
   * Applies a single transaction to the local state.
   * @param transaction Transaction to apply.
   * @param transactionProof Additional proof information.
   */
  public async applyTransaction(
    transaction: Transaction,
    transactionProof: TransactionProof
  ): Promise<void> {}

  /**
   * Checks a transaction proof. Uses local state
   * and public information (e.g. plasma blocks).
   * @param transaction Transaction to check.
   * @param transactionProof Proof to check.
   * @returns `true` if the proof is valid, `false` otherwise.
   */
  public async checkTransactionProof(
    transaction: Transaction,
    transactionProof: TransactionProof
  ): Promise<boolean> {}
}
