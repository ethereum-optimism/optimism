import { L1ToL2TransactionBatch } from './types'

/**
 * Defines the event handler interface for handling L1-to-L2 Transaction batches.
 */
export interface L1ToL2TransactionBatchListener {
  handleL1ToL2TransactionBatch(
    transactionBatch: L1ToL2TransactionBatch
  ): Promise<void>
}
