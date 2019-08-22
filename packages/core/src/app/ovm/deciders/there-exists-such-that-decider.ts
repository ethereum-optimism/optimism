import {
  Decider,
  Decision,
  ImplicationProofItem,
  Property,
  PropertyFactory,
  Quantifier,
  QuantifierResult,
  WitnessFactory,
} from '../../../types/ovm'
import { CannotDecideError } from './utils'

export interface ThereExistsSuchThatInput {
  quantifier: Quantifier
  quantifierParameters: any
  propertyFactory: PropertyFactory
}

/**
 * Decider that decides true if the provided quantifier quantifies any results that evaluate to true.
 * If not, and the quantifier quantifies all results, it'll return false, else undecided.
 */
export class ThereExistsSuchThatDecider implements Decider {
  public async decide(
    input: ThereExistsSuchThatInput,
    noCache?: boolean
  ): Promise<Decision> {
    const quantifierResult: QuantifierResult = await input.quantifier.getAllQuantified(
      input.quantifierParameters
    )

    let anyUndecided: boolean = false
    let trueDecision: Decision
    const falseDecisions: Decision[] = []
    for (const res of quantifierResult.results) {
      const prop: Property = input.propertyFactory(res)
      try {
        const decision: Decision = await prop.decider.decide(
          prop.input,
          noCache
        )
        if (decision.outcome) {
          trueDecision = decision
          break
        }
        falseDecisions.push(decision)
      } catch (e) {
        if (e instanceof CannotDecideError) {
          anyUndecided = true
        } else {
          throw e
        }
      }
    }

    return this.getDecision(
      input,
      trueDecision,
      falseDecisions,
      anyUndecided || !quantifierResult.allResultsQuantified
    )
  }

  private async checkDecision(
    input: ThereExistsSuchThatInput
  ): Promise<Decision> {
    return this.decide(input, undefined)
  }

  /**
   * Gets the Decision that results from invocation of the ThereExistsSuchThat Decider.
   *
   * @param input The input that led to the Decision
   * @param trueDecision A [possibly undefined] Decision passing this Decider to be used as proof
   * @param falseDecisions An array of false Decisions to use as justification for this Decider returning False.
   * @param undecided Whether or not some results of this Decider are undecided
   * @returns The Decision.
   */
  private getDecision(
    input: ThereExistsSuchThatInput,
    trueDecision: Decision,
    falseDecisions: Decision[],
    undecided: boolean
  ): Decision {
    if (!trueDecision && undecided) {
      throw new CannotDecideError(
        'Cannot decide ThereExistsSuchThat due to undecided Decision or not all results being quantified.'
      )
    }
    const justification: ImplicationProofItem[] = [
      {
        implication: {
          decider: this,
          input,
        },
        implicationWitness: undefined,
      },
    ]

    if (!!trueDecision) {
      justification.push(...trueDecision.justification)
    } else {
      for (const decision of falseDecisions) {
        justification.push(...decision.justification)
      }
    }

    return {
      outcome: !!trueDecision,
      justification,
    }
  }
}
