import {
  Decider,
  Decision,
  ImplicationProofItem,
  Property,
} from '../../../types/ovm'
import { CannotDecideError, handleCannotDecideError } from './utils'

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
  private static _instance: AndDecider

  public static instance(): AndDecider {
    if (!AndDecider._instance) {
      AndDecider._instance = new AndDecider()
    }
    return AndDecider._instance
  }

  public async decide(
    input: AndDeciderInput,
    witness?: undefined,
    noCache?: boolean
  ): Promise<Decision> {
    const [leftDecision, rightDecision] = await Promise.all([
      input.left.decider
        .decide(input.left.input, input.leftWitness, noCache)
        .catch(handleCannotDecideError),
      input.right.decider
        .decide(input.right.input, input.rightWitness, noCache)
        .catch(handleCannotDecideError),
    ])

    if (!!leftDecision && !leftDecision.outcome) {
      return this.getDecision(input, leftDecision)
    }
    if (!!rightDecision && !rightDecision.outcome) {
      return this.getDecision(input, rightDecision)
    }
    if (!leftDecision || !rightDecision) {
      throw new CannotDecideError(
        'One of the AND deciders could not decide, and neither decided false.'
      )
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
