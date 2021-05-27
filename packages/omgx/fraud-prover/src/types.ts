import { BigNumber } from 'ethers'
import { BaseTrie } from 'merkle-patricia-tree'

export interface StateRootBatchHeader {
  batchIndex: BigNumber
  batchRoot: string
  batchSize: BigNumber
  prevTotalElements: BigNumber
  extraData: string
}

export interface SentMessage {
  target: string
  sender: string
  message: string
  messageNonce: number
  encodedMessage: string
  encodedMessageHash: string
  parentTransactionIndex: number
  parentTransactionHash: string
}

export interface SentMessageProof {
  stateRoot: string
  stateRootBatchHeader: StateRootBatchHeader
  stateRootProof: StateRootProof
  stateTrieWitness: string | Buffer
  storageTrieWitness: string | Buffer
}

export interface StateRootProof {
  index: number
  siblings: string[]
}

export interface TransactionBatchHeader {
  batchIndex: BigNumber
  batchRoot: string
  batchSize: BigNumber
  prevTotalElements: BigNumber
  extraData: string
}

export interface TransactionProof {
  index: number
  siblings: string[]
}

export interface StateRootBatchProof {
  stateRoot: string
  stateRootBatchHeader: StateRootBatchHeader
  stateRootProof: StateRootProof
}

export interface TransactionBatchProof {
  transaction: OvmTransaction
  transactionChainElement: TransactionChainElement
  transactionBatchHeader: TransactionBatchHeader
  transactionProof: TransactionProof
}

export enum StateTransitionPhase {
  PRE_EXECUTION,
  POST_EXECUTION,
  COMPLETE,
}

export interface AccountStateProof {
  address: string
  balance: string
  nonce: string
  codeHash: string
  storageHash: string
  accountProof: string[]
  storageProof: StorageStateProof[]
}

export interface StorageStateProof {
  key: string
  value: string
  proof: string[]
}

export interface StateDiffProof {
  header: {
    number: number
    hash: string
    stateRoot: string
    timestamp: number
  }

  accountStateProofs: AccountStateProof[]
}

export interface OvmTransaction {
  blockNumber: number
  timestamp: number
  entrypoint: string
  gasLimit: number
  l1TxOrigin: string
  l1QueueOrigin: number
  data: string
}

export interface FraudProofData {
  stateDiffProof: StateDiffProof
  transactionProof: TransactionBatchProof
  preStateRootProof: StateRootBatchProof
  postStateRootProof: StateRootBatchProof

  stateTrie: BaseTrie
  storageTries: {
    [address: string]: BaseTrie
  }
}

export interface TransactionChainElement {
  isSequenced: boolean
  queueIndex?: number
  timestamp?: number
  blockNumber?: number
  txData?: string
}
