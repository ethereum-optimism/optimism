import { Address, Balances, SignedTransaction, State } from './types'

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
   * @returns The updated state resulting from the transaction
   */
  applyTransaction(signedTransaction: SignedTransaction): Promise<State>
}
