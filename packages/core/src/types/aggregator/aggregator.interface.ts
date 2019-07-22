import { BlockTransactionCommitment, Transaction } from '../serialization'

export interface Aggregator {
  /**
   * Notifies the Aggregator of the provided Transaction so it may be included in the next block.
   *
   * @param transaction the Transaction in question
   *
   * @returns the BlockTransactionCommitment indicating the transaction will be included in the next block
   */
  ingestTransaction(
    transaction: Transaction
  ): Promise<BlockTransactionCommitment>

  /**
   * Gets the public key of the Aggregator to be able to validate signatures
   *
   * @returns the public key
   */
  getPublicKey(): Promise<any>
}
