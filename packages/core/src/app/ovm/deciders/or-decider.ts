import {
  Decider,
  Decision,
  ImplicationProofItem,
  Property,
} from '../../../types/ovm'
import { CannotDecideError } from './utils'

export interface OrDeciderInput {
  properties: Property[]
  witnesses: any[]
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
    witness?: undefined,
    noCache?: boolean
  ): Promise<Decision> {
    const decisions: Decision[] = await Promise.all(
      input.properties.map((property: Property, index: number) => {
        return this.decideWithoutThrowingCannotDecide(
          property,
          input.witnesses[index],
          noCache
        )
      })
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
   * Calls decide on the provided Property's Decider with the appropriate input and catches
   * CannotDecideError, returning undefined if it occurs.
   *
   * @param property the Property with the Decider to decide and the input to pass it
   * @param witness the witness for the Decider
   * @param noCache whether or not to use the cache if one is available for previous decisions
   */
  private async decideWithoutThrowingCannotDecide(
    property: Property,
    witness: any,
    noCache: boolean
  ): Promise<Decision> {
    try {
      return await property.decider.decide(property.input, witness, noCache)
    } catch (e) {
      if (e instanceof CannotDecideError) {
        return undefined
      }
      throw e
    }
  }

  /**
   * Gets the Decision that results from invocation of the Or decider, which simply
   * returns true if any of the sub-Decisions returned true.
   *
   * @param input The input that led to the Decision
   * @param subDecision The decision of the wrapped Property, provided the witness
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
