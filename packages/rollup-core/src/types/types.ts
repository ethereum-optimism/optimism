/* External Imports */
import { BigNumber } from '@eth-optimism/core-utils'

// TODO: Probably not necessary?
//  Maybe just a map from token -> contract slot index (e.g. {ETH: 1, BAT: 2, REP: 3})?
export type TokenType = number

export interface State {}
export interface RollupBlock {
  blockNumber: number
  stateRoot: string
  transactions: string[]
}

export interface L2ToL1Message {
  nonce: BigNumber
  ovmSender: Address
  callData: string
}

/* Types */
export type Address = string
export type StorageSlot = string
export type Signature = string
export type StorageValue = string

export interface Transaction {
  ovmEntrypoint: Address
  ovmCalldata: string
}

export interface SignedTransaction {
  signature: Signature
  transaction: Transaction
}

export interface StorageElement {
  contractAddress: Address
  storageSlot: StorageSlot
  storageValue: StorageValue
}

export interface ContractStorage {
  contractAddress: Address
  contractNonce: BigNumber
  contractCode?: string
  // TODO: Add others as necessary
}

export interface TransactionLog {
  data: string
  topics: string[]
  logIndex: BigNumber
  transactionIndex: BigNumber
  transactionHash: string
  blockHash: string
  blockNumber: BigNumber
  address: Address
}

export interface TransactionReceipt {
  status: boolean
  transactionHash: string
  transactionIndex: BigNumber
  blockHash: string
  blockNumber: BigNumber
  contractAddress: Address
  cumulativeGasUsed: BigNumber
  gasUsed: BigNumber
  logs: TransactionLog[]
}

export interface TransactionResult {
  transactionNumber: BigNumber
  transactionReceipt: TransactionReceipt
  abiEncodedTransaction: string
  updatedStorage: StorageElement[]
  updatedContracts: ContractStorage[]
}
