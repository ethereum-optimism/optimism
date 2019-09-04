import { DefaultWallet, abi } from '@pigi/core'

export type UniTokenType = 'uni'
export type PigiTokenType = 'pigi'
export type TokenType = UniTokenType | PigiTokenType

export type SUCCESS = 'SUCCESS'
export type FAILURE = 'FAILURE'

export type Address = string

export interface Balances {
  [tokenType: string]: number
}

export interface Swap {
  tokenType: UniTokenType | PigiTokenType
  inputAmount: number
  minOutputAmount: number
  timeout: number
}

/* Type guard for swap transaction */
export const isSwapTransaction = (
  transaction: Transaction
): transaction is Swap => {
  return 'minOutputAmount' in transaction
}

export interface Transfer {
  tokenType: UniTokenType | PigiTokenType
  recipient: Address
  amount: number
}

/* Type guard for transfer transaction */
export const isTransferTransaction = (
  transaction: Transaction
): transaction is Transfer => {
  return 'recipient' in transaction
}

export type Transaction = Swap | Transfer

export type MockedSignature = Address

export interface SignedTransaction {
  signature: MockedSignature // For now the signature is just the address
  transaction: Transaction
}

export interface TransactionReceipt {
  aggregatorSignature: MockedSignature
  stateUpdate: any
}

export interface Storage {
  balances: Balances
}

export interface SignatureProvider {
  sign(address: string, message: string): Promise<string>
}

export interface State {
  [address: string]: Storage
}
