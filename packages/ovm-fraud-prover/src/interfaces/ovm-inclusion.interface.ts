/* External Imports */
import { BigNumber } from '@ethersproject/bignumber'

interface OVMElementInclusionProof {
  batchIndex: number | BigNumber
  indexInBatch: number | BigNumber
  siblings: string[]
}

interface StateChainBatchHeader {
  elementsMerkleRoot: string
  numElementsInBatch: number | BigNumber
  cumulativePrevElements: number | BigNumber
}

export interface OVMStateElementInclusionProof extends OVMElementInclusionProof {
  batchHeader: StateChainBatchHeader
}

interface TransactionChainBatchHeader {
  timestamp: number | BigNumber
  isL1ToL2Tx: boolean
  elementsMerkleRoot: string
  numElementsInBatch: number | BigNumber
  cumulativePrevElements: number | BigNumber
}

export interface OVMTransactionElementInclusionProof extends OVMElementInclusionProof {
  batchHeader: TransactionChainBatchHeader
}