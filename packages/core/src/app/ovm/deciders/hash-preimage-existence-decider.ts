import { Decision } from '../../../types/ovm/decider.interface'
import { DB } from '../../../types/db'
import { KeyValueStoreDecider } from './key-value-store-decider'
import { CannotDecideError, HashFunction } from './utils'

export interface HashInput {
  hash: Buffer
}

export interface PreimageWitness {
  preimage: Buffer
}

/**
 * Decider that determines whether the provided witness is the preimage to the hash in question.
 */
export class HashPreimageExistenceDecider extends KeyValueStoreDecider {
  private static readonly UNIQUE_ID = 'HashPreimageDecider'

  private readonly hashFunction: HashFunction

  constructor(db: DB, hashFunction: HashFunction) {
    super(db)

    this.hashFunction = hashFunction
  }

  public async decide(
    input: HashInput,
    witness: PreimageWitness
  ): Promise<Decision> {
    const outcome = this.hashFunction(witness.preimage).equals(input.hash)

    if (!outcome) {
      throw new CannotDecideError(
        `Preimage [${witness.preimage}] does not match hash [${input.hash}], so we cannot decide whether a preimage exists for the hash.`
      )
    }

    const decision: Decision = this.constructDecision(
      witness.preimage,
      input.hash,
      outcome
    )

    await this.storeDecision(
      input,
      HashPreimageExistenceDecider.serializeDecision(witness, input, outcome)
    )

    return decision
  }

  protected getUniqueId(): string {
    return HashPreimageExistenceDecider.UNIQUE_ID
  }

  protected deserializeDecision(decision: Buffer): Decision {
    const json: any[] = JSON.parse(decision.toString())
    return this.constructDecision(
      Buffer.from(json[0]),
      Buffer.from(json[1]),
      json[2]
    )
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

  /**
   * Creates the buffer to be stored for a Decision
   *
   * @param witness the HashPreimageWitness
   * @param input the input that led to the Decision
   * @param outcome the outcome of the Decision
   * @returns the Buffer of the serialized data
   */
  private static serializeDecision(
    witness: PreimageWitness,
    input: HashInput,
    outcome: boolean
  ): Buffer {
    return Buffer.from(
      JSON.stringify([
        witness.preimage.toString(),
        input.hash.toString(),
        outcome,
      ])
    )
  }
}
