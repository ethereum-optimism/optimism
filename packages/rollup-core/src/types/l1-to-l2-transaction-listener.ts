import { L1ToL2Transaction } from './types'

/**
 * Defines the event handler interface for L1-to-L2 Transactions.
 */
export interface L1ToL2TransactionListener {
  handleL1ToL2Transaction(transaction: L1ToL2Transaction): Promise<void>
}
