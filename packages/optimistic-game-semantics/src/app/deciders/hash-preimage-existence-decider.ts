/* External Imports */
import { HashAlgorithm } from '@pigi/core-utils'

/* Internal Imports */
import { Decider, Decision, HashPreimageDBInterface } from '../../types'
import { CannotDecideError } from './utils'

export interface HashInput {
  hash: string
}

/**
 * Decider that determines whether the provided witness is the preimage to the hash in question.
 */
export class HashPreimageExistenceDecider implements Decider {
  constructor(
    private readonly db: HashPreimageDBInterface,
    private readonly hashAlgorithm: HashAlgorithm
  ) {}

  public async decide(input: HashInput, _noCache?: boolean): Promise<Decision> {
    const preimage: string = await this.db.getPreimage(
      input.hash,
      this.hashAlgorithm
    )

    if (!preimage) {
      throw new CannotDecideError(
        `No preimage is stored for hash [${JSON.stringify(
          input
        )}], so we cannot decide whether a preimage exists for the hash.`
      )
    }

    return this.constructDecision(preimage, input.hash, true)
  }

  /**
   * Builds a Decision from the provided hash, outcome, and preimage
   *
   * @param preimage being tested
   * @param hash the hash for the Decision calculation
   * @param outcome the outcome of the Decision
   * @returns the Decision
   */
  private constructDecision(
    preimage: string,
    hash: string,
    outcome: boolean
  ): Decision {
    return {
      outcome,
      justification: [
        {
          implication: {
            decider: this,
            input: {
              hash,
            },
          },
          implicationWitness: {
            preimage,
          },
        },
      ],
    }
  }
}
