import { ethers } from 'ethers'

export interface StateRootBatchHeader {
  batchIndex: ethers.BigNumber
  batchRoot: string
  batchSize: ethers.BigNumber
  prevTotalElements: ethers.BigNumber
  extraData: string
}

export interface StateRootBatch {
  header: StateRootBatchHeader
  stateRoots: string[]
}

export interface CrossDomainMessage {
  target: string
  sender: string
  message: string
  messageNonce: number
}

export interface CrossDomainMessageProof {
  stateRoot: string
  stateRootBatchHeader: StateRootBatchHeader
  stateRootProof: {
    index: number
    siblings: string[]
  }
  stateTrieWitness: string
  storageTrieWitness: string
}

export interface CrossDomainMessagePair {
  message: CrossDomainMessage
  proof: CrossDomainMessageProof
}

export interface StateTrieProof {
  accountProof: string
  storageProof: string
}
