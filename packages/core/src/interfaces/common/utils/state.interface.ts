/* External Imports */
import BigNum = require('bn.js')

export interface StateObject {
  predicate: string
  parameters: any
}

export interface StateUpdate {
  id: {
    start: BigNum
    end: BigNum
  }
  newState: StateObject
}

export interface VerifiedStateUpdate {
  start: BigNum
  end: BigNum
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
  stateUpdate: StateUpdate
  witness: any
  block: number
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
