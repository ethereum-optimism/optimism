/* External Imports */
import {
  BlockWithTransactions,
  Provider,
  TransactionResponse,
} from '@ethersproject/abstract-provider'

export enum QueueOrigin {
  Sequencer = 0,
  L1ToL2 = 1,
}

export const queueOriginPlainText = {
  0: QueueOrigin.Sequencer,
  1: QueueOrigin.L1ToL2,
  sequencer: QueueOrigin.Sequencer,
  l1ToL2: QueueOrigin.L1ToL2,
}

/**
 * Transaction & Blocks. These are the true data-types we expect
 * from running a batch submitter.
 */
export interface L2Transaction extends TransactionResponse {
  l1BlockNumber: number
  l1TxOrigin: string
  txType: number
  queueOrigin: number
}

export interface L2Block extends BlockWithTransactions {
  stateRoot: string
  transactions: [L2Transaction]
}

/**
 * BatchElement & Batch. These are the data-types of the compressed / batched
 * block data we submit to L1.
 */
export interface BatchElement {
  stateRoot: string
  isSequencerTx: boolean
  sequencerTxType: 0
  txData: 0
  timestamp: number
  blockNumber: number
}

export type Batch = BatchElement[]
