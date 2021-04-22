/* External Imports */
import {
  BlockWithTransactions,
  Provider,
  TransactionResponse,
} from '@ethersproject/abstract-provider'

/* Internal Imports */
import { EIP155TxData, TxType } from '@eth-optimism/core-utils'

export enum QueueOrigin {
  Sequencer = 'sequencer',
  L1ToL2 = 'l1',
}

/**
 * Transaction & Blocks. These are the true data-types we expect
 * from running a batch submitter.
 */
export interface L2Transaction extends TransactionResponse {
  l1BlockNumber: number
  l1TxOrigin: string
  txType: number
  queueOrigin: string
  rawTransaction: string
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
  sequencerTxType: undefined | TxType
  rawTransaction: undefined | string
  timestamp: number
  blockNumber: number
}

export type Batch = BatchElement[]
