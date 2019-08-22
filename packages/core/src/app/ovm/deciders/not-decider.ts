import {
  Decider,
  Decision,
  ImplicationProofItem,
  Property,
} from '../../../types/ovm'

export interface NotDeciderInput {
  property: Property
}

/**
 * Decider that decides true iff the provided property evaluates to false.
 */
export class NotDecider implements Decider {
  public async decide(
    input: NotDeciderInput,
    noCache?: boolean
  ): Promise<Decision> {
    const decision: Decision = await input.property.decider.decide(
      input.property.input,
      noCache
    )

    return this.getDecision(input, decision)
  }

  /**
   * Gets the Decision that results from invocation of the Not decider, which simply
   * returns the opposite outcome than the provided Decision.
   *
   * @param input The input that led to the Decision
   * @param subDecision The decision of the wrapped Property, provided the witness
   * @returns The Decision.
   */
  private getDecision(input: NotDeciderInput, subDecision: Decision): Decision {
    const justification: ImplicationProofItem[] = [
      {
        implication: {
          decider: this,
          input,
        },
        implicationWitness: undefined,
      },
      ...subDecision.justification,
    ]

    return {
      outcome: !subDecision.outcome,
      justification,
    }
  }
}
