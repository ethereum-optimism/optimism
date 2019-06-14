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

export interface Range {
  start: BigNum
  end: BigNum
}

export interface BlockRange extends Range {
  block: BigNum
}

export interface Transaction {
  range: Range
  witness: any
  block: number
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
