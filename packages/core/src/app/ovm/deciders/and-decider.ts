import {
  Decider,
  Decision,
  ImplicationProofItem,
  Property,
} from '../../../types/ovm'

export interface AndDeciderInput {
  left: Property
  leftWitness: any
  right: Property
  rightWitness: any
}

/**
 * Decider that decides true iff both of the provided properties evaluate to true.
 */
export class AndDecider implements Decider {
  public async decide(
    input: AndDeciderInput,
    witness?: undefined,
    noCache?: boolean
  ): Promise<Decision> {
    const [leftDecision, rightDecision] = await Promise.all([
      input.left.decider.decide(input.left.input, input.leftWitness, noCache),
      input.right.decider.decide(
        input.right.input,
        input.rightWitness,
        noCache
      ),
    ])

    if (!leftDecision.outcome) {
      return this.getDecision(input, leftDecision)
    }
    if (!rightDecision.outcome) {
      return this.getDecision(input, rightDecision)
    }

    const justification: ImplicationProofItem[] = []
    if (!!leftDecision.justification.length) {
      justification.push(...leftDecision.justification)
    }
    if (!!rightDecision.justification.length) {
      justification.push(...rightDecision.justification)
    }

    return this.getDecision(input, { outcome: true, justification })
  }

  /**
   * Gets the Decision that results from invocation of the And decider, which simply
   * returns true if both sub-Decisions returned true.
   *
   * @param input The input that led to the Decision
   * @param subDecision The decision of the wrapped Property, provided the witness
   * @returns The Decision
   */
  private getDecision(input: AndDeciderInput, subDecision: Decision): Decision {
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
      outcome: subDecision.outcome,
      justification,
    }
  }
}
