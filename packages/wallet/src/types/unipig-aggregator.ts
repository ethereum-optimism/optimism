import { Address, SignedStateReceipt, SignedTransaction } from './types'

export interface UnipigAggregator {
  getPendingBlockNumber(): number
  getNextTransitionIndex(): number

  /**
   * Gets the State for the provided address if State exists.
   *
   * @param address The address in question
   * @returns The SignedStateReceipt containing the state and the aggregator
   * guarantee that it exists. If it does not exist, this will include the
   * aggregator guarantee that it does not exist.
   */
  getState(address: Address): Promise<SignedStateReceipt>

  /**
   * Handles the provided transaction and returns the updated state and block and
   * transition in which it will be updated, guaranteed by the aggregator's signature.
   *
   * @param signedTransaction The transaction to apply
   * @returns The SignedTransactionReceipt
   */
  applyTransaction(
    signedTransaction: SignedTransaction
  ): Promise<SignedStateReceipt[]>

  /**
   * Requests faucet funds on behalf of the sender and returns the updated
   * state resulting from the faucet allocation, including the guarantee that
   * it will be included in a specific block and transition.
   *
   * @param signedTransaction The faucet transaction
   * @returns The SignedTransactionReceipt
   */
  requestFaucetFunds(
    signedTransaction: SignedTransaction
  ): Promise<SignedStateReceipt>

  /**
   * Gets the total number of transactions processed by this aggregator.
   *
   * @returns The transaction count.
   */
  getTransactionCount(): Promise<number>
}
