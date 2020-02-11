/* External Imports */
import { BigNumber } from '@eth-optimism/core-utils'
import {
  Address,
  StorageSlot,
  StorageValue,
  Transaction,
  TransactionResult,
} from '@eth-optimism/rollup-core'

export interface RollupStateMachine {
  /**
   * Gets the state for the provided address, if one exists.
   *
   * @param targetContract The contract being retrieved.
   * @param targetStorageSlot The slot to the storage being retrieved.
   * @returns The storage value at the specified contract & key.
   */
  getStorageAt(
    targetContract: Address,
    targetStorageSlot: StorageSlot
  ): Promise<StorageValue>

  /**
   * Applies the provided signed transaction, adjusting balances accordingly.
   *
   * @param abiEncodedTransaction The ABI-encoded transaction to execute.
   * @returns The TransactionResult resulting from the transaction
   */
  applyTransaction(abiEncodedTransaction: string): Promise<TransactionResult>

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
