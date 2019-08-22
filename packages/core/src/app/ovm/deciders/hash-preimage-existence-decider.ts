import { Decider, Decision, HashPreimageDbInterface } from '../../../types/ovm'
import { CannotDecideError } from './utils'
import { HashAlgorithm } from '../../../types/utils'

export interface HashInput {
  hash: Buffer
}

/**
 * Decider that determines whether the provided witness is the preimage to the hash in question.
 */
export class HashPreimageExistenceDecider implements Decider {
  constructor(
    private readonly db: HashPreimageDbInterface,
    private readonly hashAlgorithm: HashAlgorithm
  ) {}

  public async decide(input: HashInput, _noCache?: boolean): Promise<Decision> {
    const preimage: Buffer = await this.db.getPreimage(
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
    preimage: Buffer,
    hash: Buffer,
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
