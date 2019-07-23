export interface ProofElement {
  property: Property
  witness: {}
}

export type Proof = ProofElement[]

export interface Property {
  decider: Decider
  input: {}
}

export type PropertyFactory = (input: any) => Property

export interface ImplicationProofElement {
  implication: Property
  implicationWitness: any
}

export interface Decision {
  outcome: boolean
  implicationProof: ImplicationProofElement[] // constructed such that claim[N] --> claim[N-1] --> claim[N-2]... Claim[0]
}

export const UNDECIDED = undefined
export type Undecided = undefined
export type DecisionStatus = boolean | Undecided

/**
 * Defines the Decider interface that implementations capable of making decisions
 * on the provided input according to the logic of the specific implementation.
 *
 * For example: A PreimageExistsDecider would be able to make decisions on whether
 * or not the provided _witness, when hashed, results in the provided _input.
 */
export interface Decider {
  /**
   * Makes a Decision on the provided input
   * @param _input
   * @param _witness
   */
  decide(_input: any, _witness: any): Decision

  /**
   * Checks whether or not a decision has been made for the provided Input
   * Note: This should access a cache decisions that have been made
   * @param _input
   * @returns the DecisionStatus, indicating if one was made.
   */
  checkDecision(_input: any): DecisionStatus
}
