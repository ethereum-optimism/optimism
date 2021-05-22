import {
  EnqueueEntry,
  StateRootBatchEntry,
  StateRootEntry,
  TransactionBatchEntry,
  TransactionEntry,
} from './database-types'

export type EnqueueResponse = EnqueueEntry & {
  ctcIndex: number | null
}

export interface TransactionResponse {
  batch: TransactionBatchEntry
  transaction: TransactionEntry
}

export interface TransactionBatchResponse {
  batch: TransactionBatchEntry
  transactions: TransactionEntry[]
}

export interface StateRootResponse {
  batch: StateRootBatchEntry
  stateRoot: StateRootEntry
}

export interface StateRootBatchResponse {
  batch: StateRootBatchEntry
  stateRoots: StateRootEntry[]
}

export interface ContextResponse {
  blockNumber: number
  timestamp: number
  blockHash: string
}

export interface GasPriceResponse {
  gasPrice: string
}

export type SyncingResponse =
  | {
      syncing: true
      highestKnownTransactionIndex: number
      currentTransactionIndex: number
    }
  | {
      syncing: false
      currentTransactionIndex: number
    }
