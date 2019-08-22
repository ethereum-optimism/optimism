export interface ProofItem {
  property: Property
  witness: {}
}

export type Proof = ProofItem[]

export interface Property {
  decider: Decider
  input: {}
  witness?: any
}

export type PropertyFactory = (input: any) => Property
export type WitnessFactory = (input: any) => any

export interface ImplicationProofItem {
  implication: Property
  implicationWitness: any
}

export interface Decision {
  outcome: boolean
  justification: ImplicationProofItem[] // constructed such that claim[N] --> claim[N-1] --> claim[N-2]... Claim[0]
}

/**
 * Defines the Decider interface that implementations capable of making decisions
 * on the provided input according to the logic of the specific implementation.
 *
 * For example: A PreimageExistsDecider would be able to make decisions on whether
 * or not a hash preimage has been stored in its database.
 */
export interface Decider {
  /**
   * Makes a Decision on the provided input.
   *
   * If this Decider is capable of caching and the noCache flag is not set,
   * it will first check to see if a decision has already been made on this input.
   * If this Decider is capable of caching and the noCache flag is set,
   * it will make the Decision, if possible, and overwrite the cache.
   *
   * @param input The input on which a decision is being made
   * @param noCache [optional] Flag set when caching should not be used.
   * @returns the Decision that was made if one was possible
   * @throws CannotDecideError if it cannot decide.
   */
  decide(input: any, noCache?: boolean): Promise<Decision>
}
