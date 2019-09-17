import {
  Address,
  Balances,
  SignedTransaction,
  State,
  StateSnapshot,
  StateUpdate,
} from './types'

export interface RollupStateMachine {
  /**
   * Gets the state for the provided address, if one exists.
   *
   * @param account The account in question
   * @returns The StateSnapshot object with the state and the inclusion proof
   */
  getState(account: Address): Promise<StateSnapshot>

  /**
   * Applies the provided signed transaction, adjusting balances accordingly.
   *
   * @param signedTransaction The signed transaction to execute.
   * @returns The StateUpdate updated state resulting from the transaction
   */
  applyTransaction(signedTransaction: SignedTransaction): Promise<StateUpdate>

  /**
   * Atomically applies the provided signed transactions,
   * adjusting balances accordingly and returning a single StateUpdate.
   * Transactions are guaranteed to be applied sequentially.
   *
   * @param signedTransactions The signed transactions to execute.
   * @returns The StateUpdate updated state resulting from the transactions
   */
  applyTransactions(
    signedTransactions: SignedTransaction[]
  ): Promise<StateUpdate>
}
