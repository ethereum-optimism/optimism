export interface DecodedBatchTransaction {
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

export interface IndexedEntry {
  index: number
  key: string
}

export interface EnqueueTransactionEntry extends IndexedEntry {
  target: string
  data: string
  gasLimit: string
  origin: string
  blockNumber: number
  timestamp: number
  ctcIndex: number | null
}

export interface BatchTransactionEntry extends IndexedEntry {
  batchIndex: number
  blockNumber: number
  timestamp: number
  gasLimit: string
  target: string
  origin: string
  data: string
  value: string
  queueOrigin: 'sequencer' | 'l1'
  queueIndex: number | null
  decoded: DecodedBatchTransaction | null
}
