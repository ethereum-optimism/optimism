import {
  Decider,
  Decision,
  ImplicationProofItem,
  Property,
} from '../../../types/ovm'
import { CannotDecideError, handleCannotDecideError } from './utils'

export interface OrDeciderInput {
  properties: Property[]
}

/**
 * Decider that decides true if any of the provided properties evaluate to true.
 */
export class OrDecider implements Decider {
  private static _instance: OrDecider

  public static instance(): OrDecider {
    if (!OrDecider._instance) {
      OrDecider._instance = new OrDecider()
    }
    return OrDecider._instance
  }

  public async decide(
    input: OrDeciderInput,
    noCache?: boolean
  ): Promise<Decision> {
    const decisions: Decision[] = await Promise.all(
      input.properties.map((property: Property, index: number) =>
        property.decider
          .decide(property.input, noCache)
          .catch(handleCannotDecideError)
      )
    )

    let trueDecision: Decision
    let cannotDecide: boolean = false
    const falseJustifications: ImplicationProofItem[] = []
    for (const decision of decisions) {
      if (!decision) {
        cannotDecide = true
        continue
      }
      if (decision.outcome) {
        trueDecision = decision
        break
      } else {
        falseJustifications.push(...decision.justification)
      }
    }

    if (trueDecision) {
      return this.getDecision(input, trueDecision)
    }

    if (cannotDecide) {
      throw new CannotDecideError(
        'At least one of the OR deciders could not decide and none returned true, so this cannot be decided.'
      )
    }

    return this.getDecision(input, {
      outcome: false,
      justification: falseJustifications,
    })
  }

  /**
   * Gets the Decision that results from invocation of the Or decider, which simply
   * returns true if any of the sub-Decisions returned true.
   *
   * @param input The input that led to the Decision
   * @param subDecision The decision of the wrapped Property
   * @returns The Decision
   */
  private getDecision(input: OrDeciderInput, subDecision: Decision): Decision {
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
