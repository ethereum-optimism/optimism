import { TransactionOutput } from '../types'

export enum QueueOrigin {
  L1_TO_L2_QUEUE = 0,
  SAFETY_QUEUE = 1,
  SEQUENCER = 2,
}

export enum BatchSubmissionStatus {
  QUEUED = 'QUEUED',
  SUBMITTING = 'SUBMITTING',
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

export interface BatchSubmission {
  batchNumber: number
  startIndex?: number
  status: BatchSubmissionStatus
  submissionTxHash: string
}

export interface TransactionBatchSubmission extends BatchSubmission {
  transactions: TransactionOutput[]
}

export interface StateCommitmentBatchSubmission extends BatchSubmission {
  stateRoots: string[]
}
