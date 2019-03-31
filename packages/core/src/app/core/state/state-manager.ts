import {
  StateManager,
  Transaction,
  TransactionProof,
} from '../../../interfaces'

export class PGStateManager implements StateManager {
  applyTransaction(
    transaction: Transaction,
    transactionProof: TransactionProof
  ): Promise<void> {}

  checkTransactionProof(
    transaction: Transaction,
    transactionProof: TransactionProof
  ): Promise<boolean> {}
}
