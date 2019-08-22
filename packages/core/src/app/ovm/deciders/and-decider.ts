import {
  Decider,
  Decision,
  ImplicationProofItem,
  Property,
} from '../../../types/ovm'
import { CannotDecideError, handleCannotDecideError } from './utils'

export interface AndDeciderInput {
  properties: Property[]
}

/**
 * Decider that decides true iff both of the provided properties evaluate to true.
 */
export class AndDecider implements Decider {
  private static _instance: AndDecider

  public static instance(): AndDecider {
    if (!AndDecider._instance) {
      AndDecider._instance = new AndDecider()
    }
    return AndDecider._instance
  }

  public async decide(
    input: AndDeciderInput,
    noCache?: boolean
  ): Promise<Decision> {
    const decisions: Decision[] = await Promise.all(
      input.properties.map((p) =>
        p.decider.decide(p.input, noCache).catch(handleCannotDecideError)
      )
    )

    const justification: ImplicationProofItem[] = []
    let undecideable = false
    let falseDecision
    for (const decision of decisions) {
      if (!decision) {
        undecideable = true
        continue
      }

      if (!decision.outcome) {
        falseDecision = decision
        break
      }
      justification.push(...decision.justification)
    }

    if (!!falseDecision) {
      return this.getDecision(input, falseDecision)
    }

    if (undecideable) {
      throw new CannotDecideError(
        'One of the AND deciders could not decide, and none decided false.'
      )
    }

    return this.getDecision(input, { outcome: true, justification })
  }

  /**
   * Gets the Decision that results from invocation of the And decider, which simply
   * returns true if both sub-Decisions returned true.
   *
   * @param input The input that led to the Decision
   * @param subDecision The decision of the wrapped Property
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
