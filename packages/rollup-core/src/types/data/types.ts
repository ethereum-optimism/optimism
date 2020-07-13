import { TransactionAndRoot } from '../types'

export enum QueueOrigin {
  L1_TO_L2_QUEUE = 0,
  SAFETY_QUEUE = 1,
  SEQUENCER = 2,
}

export enum L2BatchStatus {
  UNBATCHED = 'UNBATCHED',
  BATCHED = 'BATCHED',
  TXS_SUBMITTED = 'TXS_SUBMITTED',
  TXS_CONFIRMED = 'TXS_CONFIRMED',
  ROOTS_SUBMITTED = 'ROOTS_SUBMITTED',
  ROOTS_CONFIRMED = 'ROOTS_CONFIRMED',
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
