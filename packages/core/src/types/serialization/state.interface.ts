/* External Imports */
import { Range } from '../../types'
import { BigNumber } from '../../app/utils'

export interface StateObject {
  predicateAddress: string
  data: any
}

export interface StateUpdate {
  range: Range
  stateObject: StateObject
  depositAddress: string
  plasmaBlockNumber: BigNumber
}

export interface VerifiedStateUpdate {
  range: Range
  verifiedBlockNumber: BigNumber
  stateUpdate: StateUpdate
}

// TODO: Define this properly if not `string`. Just adding it to be able to define StateQuery.
export type Expression = string

export interface StateQuery {
  depositAddress: string
  predicateAddress: string
  start?: BigNumber
  end?: BigNumber
  method: string
  params: string[]
  filter?: Expression
}

export interface StateQueryResult {
  stateUpdate: StateUpdate
  result: string[]
}

export interface Transaction {
  depositAddress: string
  range: Range
  body: any
}

export interface TransactionResult {
  stateUpdate: StateUpdate
  validRanges: Range[]
}

export interface BlockTransaction {
  blockNumber: BigNumber
  transaction: Transaction
}

export interface BlockTransactionCommitment {
  blockTransaction: BlockTransaction
  witness: any
}

export interface OwnershipBody {
  newState: StateObject
  originBlock: BigNumber
  maxBlock: BigNumber
}

export interface OwnershipStateData {
  owner: string
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
