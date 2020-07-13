import { BigNumber } from 'ethers/utils'

interface OVMElementInclusionProof {
  batchIndex: BigNumber
  indexInBatch: BigNumber
  siblings: string[]
}

interface StateChainBatchHeader {
  elementsMerkleRoot: string
  numElementsInBatch: BigNumber
  cumulativePrevElements: BigNumber
}

export interface OVMStateElementInclusionProof extends OVMElementInclusionProof {
  batchHeader: StateChainBatchHeader
}

interface TransactionChainBatchHeader {
  timestamp: BigNumber
  isL1ToL2Tx: boolean
  elementsMerkleRoot: string
  numElementsInBatch: BigNumber
  cumulativePrevElements: BigNumber
}

export interface OVMTransactionElementInclusionProof extends OVMElementInclusionProof {
  batchHeader: TransactionChainBatchHeader
}