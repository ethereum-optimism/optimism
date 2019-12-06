/* External Imports */
import { BigNumber } from '@pigi/core-utils'

export type TokenType = number
export type Address = string

export type Signature = string

export interface Balances {
  [tokenType: number]: number
}

export interface Transfer {
  sender: Address
  recipient: Address
  // TODO: TokenType will probably actually be a reference to an L2 ERC-20 contract
  tokenType: TokenType
  amount: number
}

export interface GenericTransaction {
  sender: Address
  body: {}
}

export type RollupTransaction = Transfer | GenericTransaction

export interface SignedTransaction {
  signature: Signature
  transaction: RollupTransaction
}

export interface State {}

export interface TransactionStorage {
  contractSlotIndex: number
  storageSlotIndex: number
  storage: string
}

export interface TransactionResult {
  transactionNumber: BigNumber
  signedTransaction: SignedTransaction
  modifiedStorage: TransactionStorage[]
}

export interface RollupBlock {
  blockNumber: number
  stateRoot: string
  signedTransactions: SignedTransaction[]
}

export const isTransferTransaction = (
  transaction: RollupTransaction
): transaction is Transfer => {
  return !('body' in transaction)
}
