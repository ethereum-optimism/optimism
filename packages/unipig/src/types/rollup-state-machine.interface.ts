import { Address, SignedTransaction, StateSnapshot, StateUpdate } from './types'

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
   * adjusting balances accordingly and returning the resulting StateUpdates.
   * Transactions are guaranteed to be applied sequentially.
   *
   * @param signedTransactions The signed transactions to execute.
   * @returns The StateUpdates updated states resulting from the
   * transactions in the order in which they were passed to this function.
   */
  applyTransactions(
    signedTransactions: SignedTransaction[]
  ): Promise<StateUpdate[]>
}
