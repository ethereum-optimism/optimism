import { BigNumber } from 'ethers'

export interface StateBatchHeader {
  batchIndex: BigNumber
  batchRoot: string
  batchSize: BigNumber
  prevTotalElements: BigNumber
  extraData: string
}

export interface SentMessage {
  target: string
  sender: string
  data: string
  nonce: number
  calldata: string
  hash: string
  height: number
}

export interface MessageProof {
  stateRoot: string
  stateRootBatchHeader: StateBatchHeader
  stateRootProof: {
    index: number
    siblings: string[]
  }
  stateTrieWitness: string | Buffer
  storageTrieWitness: string | Buffer
}
