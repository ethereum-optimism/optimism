import {
  HistoryManager,
  Transaction,
  TransactionProof,
} from '../../../interfaces'

export class PGHistoryManager implements HistoryManager {
  addTransactions(transactions: Transaction[]): Promise<void> {}

  getTransactionProof(transaction: Transaction): Promise<TransactionProof> {}
}
