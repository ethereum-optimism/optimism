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

export interface StateQuery {
  // TODO: Define this
}

export interface StateQueryResult {
  // TODO: Define this
}

export interface Transaction {
  stateUpdate: StateUpdate
  witness: any
  block: number
}

export interface HistoryProof {
  // TODO: Define this
}

export type InclusionProof = string[]

export interface ProofElementDeposit {
  transaction: Transaction
}

export interface ProofElementTransaction {
  transaction: Transaction
  inclusionProof: InclusionProof
}

export type ProofElement = ProofElementDeposit | ProofElementTransaction

export type TransactionProof = ProofElement[]
