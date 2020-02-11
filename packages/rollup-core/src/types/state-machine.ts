/* External Imports */
import { BigNumber } from '@eth-optimism/core-utils'

/* Internal Imports */
import { SignedTransaction, State, TransactionResult } from './types'

export interface RollupStateMachine {
  /**
   * Gets the state for the provided address, if one exists.
   *
   * @param slotIndex The slot slot index of the account in question.
   * @returns The StateSnapshot object with the state.
   */
  getState(slotIndex: string): Promise<State>

  /**
   * Applies the provided signed transaction, adjusting balances accordingly.
   *
   * @param signedTransaction The signed transaction to execute.
   * @returns The TransactionResult resulting from the transaction
   */
  applyTransaction(
    signedTransaction: SignedTransaction
  ): Promise<TransactionResult>

  /**
   * Gets all TransactionResults processed by this State Machine since (after) the provided
   * transaction number.
   *
   * @param transactionNumber The transaction number in question
   * @returns the ordered list of transactions since the provided transaction number
   */
  getTransactionResultsSince(
    transactionNumber: BigNumber
  ): Promise<TransactionResult[]>
}
