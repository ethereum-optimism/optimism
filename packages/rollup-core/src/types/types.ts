/* External Imports */
import { BigNumber } from '@eth-optimism/core-utils'

import { Log, TransactionResponse } from 'ethers/providers/abstract-provider'

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
  nonce: number
  ovmSender: Address
  callData: string
}

export interface RollupTransaction {
  nonce: number
  gasLimit: number
  sender: Address
  target: Address
  calldata: string
  gasLimit: string
}

export type L1Batch = RollupTransaction[]
export interface BlockBatches {
  timestamp: number
  blockNumber: number
  batches: L1Batch[]
}

export type L1BatchParser = (
  l: Log,
  transaction: TransactionResponse
) => Promise<L1Batch>

export interface BatchLogParserContext {
  topic: string
  contractAddress: Address
  parseL1Batch: L1BatchParser
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
