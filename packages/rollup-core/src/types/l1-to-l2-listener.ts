import {
  L1ToL2StateCommitmentBatch,
  L1ToL2Transaction,
  L1ToL2TransactionBatch,
} from './types'

/**
 * Defines the event handler interface for L1-to-L2 Transactions.
 */
export interface L1ToL2TransactionListener {
  handleL1ToL2Transaction(transaction: L1ToL2Transaction): Promise<void>
}

export interface L1ToL2TransactionBatchListener {
  handleTransactionBatch(batch: L1ToL2TransactionBatch): Promise<void>
}

export interface L1ToL2StateCommitmentBatchHandler {
  handleStateCommitmentBatch(batch: L1ToL2StateCommitmentBatch): Promise<void>
}
