import {
  Address,
  Balances,
  SignedTransaction,
  State,
  StateUpdate,
} from './types'

export interface RollupStateMachine {
  /**
   * Gets the balances for the provided address.
   * This should never be undefined, instead returning zero balances when missing.
   *
   * @param account The account in question
   * @returns The Balances object
   */
  getBalances(account: Address): Promise<Balances>

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
