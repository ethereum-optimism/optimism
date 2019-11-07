/* External Imports */
import { areEqual } from '@pigi/core-utils'

/* Internal Imports */
import { Decider, Decision, ImplicationProofItem } from '../../types'

export interface EqualityDeciderInput {
  itemsToCompare: any[]
}

/**
 * Decider that decides true if all of the provided values are equal.
 */
export class EqualityDecider implements Decider {
  private static _instance: EqualityDecider

  public static instance(): EqualityDecider {
    if (!EqualityDecider._instance) {
      EqualityDecider._instance = new EqualityDecider()
    }
    return EqualityDecider._instance
  }

  public async decide(
    input: EqualityDeciderInput,
    noCache?: boolean
  ): Promise<Decision> {
    if (input.itemsToCompare && input.itemsToCompare.length) {
      const compareTo: any = input.itemsToCompare[0]
      for (const item of input.itemsToCompare) {
        if (!areEqual(compareTo, item)) {
          return this.getDecision(input, [compareTo, item])
        }
      }
    }

    return this.getDecision(input)
  }

  /**
   * Gets the Decision that results from invocation of the Equality decider, which simply
   * returns true if all presented items are equal
   *
   * @param input The input that led to the Decision
   * @param mismatchedItems The items that differ, proving that the input is not all equal
   * @returns The Decision
   */
  private getDecision(
    input: EqualityDeciderInput,
    mismatchedItems?: any[]
  ): Decision {
    const justification: ImplicationProofItem[] = [
      {
        implication: {
          decider: this,
          input,
        },
        implicationWitness: mismatchedItems,
      },
    ]

    return {
      outcome: !mismatchedItems,
      justification,
    }
  }
}
