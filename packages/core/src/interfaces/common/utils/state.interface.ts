/* External Imports */
import BigNum = require('bn.js')
import { Range } from './range-store.interface'

export interface StateObject {
  predicate: string
  parameters: any
}

export interface StateUpdate {
  range: Range
  stateObject: StateObject
  depositContract: string
  plasmaBlockNumber: number
}

export interface VerifiedStateUpdate {
  range: Range
  verifiedBlockNumber: number
  stateUpdate: StateUpdate
}

// TODO: Define this properly if not `string`. Just adding it to be able to define StateQuery.
export type Expression = string

export interface StateQuery {
  plasmaContract: string
  predicateAddress: string
  start?: BigNum
  end?: BigNum
  method: string
  params: string[]
  filter?: Expression
}

export interface StateQueryResult {
  stateUpdate: StateUpdate
  result: string[]
}

export interface Transaction {
  depositContract: string
  methodId: string
  parameters: any
  range: Range
}

export type InclusionProof = string[]
export type ExclusionProof = string[]

export interface ProofElementDeposit {
  transaction: Transaction
}

export interface ProofElementTransaction {
  transaction: Transaction
  inclusionProof: InclusionProof
}

export interface ProofElementTransactionExclusion {
  transaction: Transaction
  exclusionProof: ExclusionProof
}

export type ProofElement = ProofElementDeposit | ProofElementTransaction

export type TransactionProof = ProofElement[]

export type HistoryProof = Array<
  ProofElementDeposit | ProofElementTransaction | ExclusionProof
>
