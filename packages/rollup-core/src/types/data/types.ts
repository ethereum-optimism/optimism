import {TransactionAndRoot} from '../types'

export enum L2BatchStatus {
  UNBATCHED = 'UNBATCHED',
  BATCHED = 'BATCHED',
  TXS_SUBMITTED = 'TXS_SUBMITTED',
  TXS_CONFIRMED = 'TXS_CONFIRMED',
  ROOTS_SUBMITTED = 'ROOTS_SUBMITTED',
  ROOTS_CONFIRMED = 'ROOTS_CONFIRMED'
}

export interface L1BatchRecord {
  blockTimestamp: number
  batchNumber: number
  batchSize: number
}

export interface L1BatchSubmission {
  l1TxBatchTxHash: string
  l1StateRootBatchTxHash: string
  status: string
  l2BatchNumber: number
  transactions: TransactionAndRoot[]
}