import { TransactionOutput } from '../types'

export enum QueueOrigin {
  L1_TO_L2_QUEUE = 0,
  SAFETY_QUEUE = 1,
  SEQUENCER = 2,
}

export enum OptimisticCanonicalChainStatus {
  QUEUED = 'QUEUED',
  SENT = 'SENT',
  FINALIZED = 'FINALIZED',
}

export enum GethSubmissionQueueStatus {
  QUEUED = 'QUEUED',
  SENT = 'SENT',
}

export enum VerificationStatus {
  UNVERIFIED = 'UNVERIFIED',
  VERIFIED = 'VERIFIED',
  FRAUDULENT = 'FRAUDULENT',
  REMOVED = 'REMOVED',
}

export interface GethSubmissionRecord {
  blockTimestamp: number
  submissionNumber: number
  size: number
}

export interface OccBatchSubmission {
  l1TxBatchTxHash: string
  l1StateRootBatchTxHash: string
  txBatchStatus: OptimisticCanonicalChainStatus
  rootBatchStatus: OptimisticCanonicalChainStatus
  occBatchNumber: number
  transactions: TransactionOutput[]
}
