/* External Imports */
import {
  BlockWithTransactions,
  TransactionResponse,
} from '@ethersproject/abstract-provider'

/**
 * Structure of the response returned by L2Geth nodes when querying the `rollup_getInfo` endpoint.
 */
export interface RollupInfo {
  mode: 'sequencer' | 'verifier'
  syncing: boolean
  ethContext: {
    blockNumber: number
    timestamp: number
  }
  rollupContext: {
    index: number
    queueIndex: number
  }
}

/**
 * Enum used for the two transaction types (queue and direct to Sequencer).
 */
export enum QueueOrigin {
  Sequencer = 'sequencer',
  L1ToL2 = 'l1',
}

/**
 * JSON transaction representation when returned by L2Geth nodes. This is simply an extension to
 * the standard transaction response type. You do NOT need to use this type unless you care about
 * having typed access to L2-specific fields.
 */
export interface L2Transaction extends TransactionResponse {
  l1BlockNumber: number
  l1TxOrigin: string
  queueOrigin: string
  rawTransaction: string
}

/**
 * JSON block representation when returned by L2Geth nodes. Just a normal block but with
 * L2Transaction objects instead of the standard transaction response object.
 */
export interface L2Block extends BlockWithTransactions {
  stateRoot: string
  transactions: [L2Transaction]
}

/**
 * Generic batch element, either a state root batch element or a transaction batch element.
 */
export interface BatchElement {
  // Only exists on state root batch elements.
  stateRoot: string

  // Only exists on transaction batch elements.
  isSequencerTx: boolean
  rawTransaction: undefined | string

  // Batch element context, exists on all batch elements.
  timestamp: number
  blockNumber: number
}

/**
 * List of batch elements.
 */
export type Batch = BatchElement[]
