/* External Imports */
import { BigNumber } from '@pigi/core-utils'

/* Types */
export type Address = string
export type StorageSlot = string
export type Signature = string
export type StorageValue = string

export interface Transaction {
  ovmEntrypoint: Address
  ovmCalldata: string
}

export interface StorageElement {
  contractAddress: Address
  storageSlot: StorageSlot
  storageValue: StorageValue
}

export interface TransactionResult {
  transactionNumber: BigNumber
  transaction: Transaction
  updatedStorage: StorageElement[]
  // This should include nonces, new contract code, maybe even timestamp / queue origin
}
