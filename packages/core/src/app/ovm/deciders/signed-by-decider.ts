import { Decision } from '../../../types/ovm'
import { KeyValueStoreDecider } from './key-value-store-decider'
import { DB } from '../../../types/db'
import { CannotDecideError } from './utils'

export interface SignedByInput {
  publicKey: Buffer
  message: Buffer
}

export interface SignedByWitness {
  signature: Buffer
}

export type SignatureVerifier = (
  publicKey: Buffer,
  message: Buffer,
  signature: Buffer
) => Promise<boolean>

/**
 * Decider that determines whether the provided witness is the provided message signed by
 * the private key associated with the provided public key.
 */
export class SignedByDecider extends KeyValueStoreDecider {
  private static readonly UNIQUE_ID = 'SignedByDecider'

  private readonly signatureVerifier: SignatureVerifier

  constructor(db: DB, signatureVerifier: SignatureVerifier) {
    super(db)

    this.signatureVerifier = signatureVerifier
  }

  public async makeDecision(
    input: SignedByInput,
    witness: SignedByWitness
  ): Promise<Decision> {
    const signatureMatches: boolean =
      witness &&
      (await this.signatureVerifier(
        input.publicKey,
        input.message,
        witness.signature
      ))

    if (!signatureMatches) {
      throw new CannotDecideError(
        'Signature does not match the provided witness, but we do not know for certain that the message was not signed by the private key associated with the provided public key.'
      )
    }

    await this.storeDecision(
      input,
      SignedByDecider.serializeDecision(witness, input, signatureMatches)
    )

    return this.constructDecision(
      witness.signature,
      input.publicKey,
      input.message,
      signatureMatches
    )
  }

  protected getUniqueId(): string {
    return SignedByDecider.UNIQUE_ID
  }

  protected deserializeDecision(decision: Buffer): Decision {
    const json: any[] = JSON.parse(decision.toString())
    return this.constructDecision(
      Buffer.from(json[0]),
      Buffer.from(json[1]),
      Buffer.from(json[2]),
      json[3]
    )
  }

  /**
   * Builds a Decision from the provided signature, public key, message, and outcome.
   *
   * @param signature The signature
   * @param publicKey The public key used with the signature
   * @param message The decrypted message
   * @param outcome the outcome of the Decision
   * @returns the Decision
   */
  private constructDecision(
    signature: Buffer,
    publicKey: Buffer,
    message: Buffer,
    outcome: boolean
  ): Decision {
    return {
      outcome,
      justification: [
        {
          implication: {
            decider: this,
            input: {
              publicKey,
              message,
            },
          },
          implicationWitness: {
            signature,
          },
        },
      ],
    }
  }

  /**
   * Creates the buffer to be stored for a Decision
   *
   * @param witness the SignedByWitness
   * @param input the input that led to the Decision
   * @param outcome the outcome of the Decision
   * @returns the Buffer of the serialized data
   */
  private static serializeDecision(
    witness: SignedByWitness,
    input: SignedByInput,
    outcome: boolean
  ): Buffer {
    return Buffer.from(
      JSON.stringify([
        witness.signature.toString(),
        input.publicKey.toString(),
        input.message.toString(),
        outcome,
      ])
    )
  }
}
