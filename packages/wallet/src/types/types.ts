import { InclusionProof } from '@pigi/core'

export type UniTokenType = 'uni'
export type PigiTokenType = 'pigi'
export type TokenType = UniTokenType | PigiTokenType

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

export interface FaucetRequest {
  requester: Address
  // Todo: might want to change this to token -> amount map
  amount: number
}

export const isFaucetTransaction = (
  transaction: Transaction
): transaction is FaucetRequest => {
  return 'requester' in transaction
}

export type Transaction = Swap | Transfer | FaucetRequest

export type Signature = string

export interface SignedTransaction {
  signature: Signature
  transaction: Transaction
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

export type InclusionProof = string[]

export interface StateInclusionProof {
  [address: string]: InclusionProof
}

export interface StateUpdate {
  transactions: SignedTransaction[]
  startRoot: string
  endRoot: string
  updatedState: State
  updatedStateInclusionProof: StateInclusionProof
}

export interface RollupTransition {
  number: number
  blockNumber: number
  transactions: SignedTransaction[]
  startRoot: string
  endRoot: string
}

export interface RollupBlock {
  number: number
  transitions: RollupTransition[]
}

export interface TransactionReceipt {
  blockNumber: number
  transitionIndex: number
  transaction: SignedTransaction
  startRoot: string
  endRoot: string
  updatedState: State
  updatedStateInclusionProof: StateInclusionProof
}

export interface SignedTransactionReceipt {
  transactionReceipt: TransactionReceipt
  signature: Signature
}

export interface StateSnapshot {
  address: string
  state: State
  stateRoot: string
  inclusionProof: InclusionProof
}

export interface StateReceipt extends StateSnapshot {
  blockNumber: number
  transitionIndex: number
}

export interface SignedStateReceipt {
  stateReceipt: StateReceipt
  signature: Signature
}
