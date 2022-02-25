export interface DecodedSequencerBatchTransaction {
  sig: {
    r: string
    s: string
    v: number
  }
  value: string
  gasLimit: string
  gasPrice: string
  nonce: string
  target: string
  data: string
}

export interface EnqueueEntry {
  index: number
  target: string
  data: string
  gasLimit: string
  origin: string
  blockNumber: number
  timestamp: number
}

export interface TransactionEntry {
  index: number
  batchIndex: number
  data: string
  blockNumber: number
  timestamp: number
  gasLimit: string
  target: string
  origin: string
  value: string
  queueOrigin: 'sequencer' | 'l1'
  queueIndex: number | null
  decoded: DecodedSequencerBatchTransaction | null
  confirmed: boolean
}

interface BatchEntry {
  index: number
  blockNumber: number
  timestamp: number
  submitter: string
  size: number
  root: string
  prevTotalElements: number
  extraData: string
  l1TransactionHash: string
  type: string
}

export type TransactionBatchEntry = BatchEntry
export type StateRootBatchEntry = BatchEntry

export interface StateRootEntry {
  index: number
  batchIndex: number
  value: string
  confirmed: boolean
}
