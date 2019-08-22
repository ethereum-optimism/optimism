import { Decider, Decision } from '../../../types/ovm'
import { CannotDecideError } from './utils'
import { SignedByDBInterface } from '../../../types/ovm/db/signed-by-db.interface'

export interface SignedByInput {
  publicKey: Buffer
  message: Buffer
}

/**
 * Decider that determines whether the provided witness is the provided message signed by
 * the private key associated with the provided public key.
 */
export class SignedByDecider implements Decider {
  constructor(
    private readonly signedByDb: SignedByDBInterface,
    private readonly myAddress: Buffer
  ) {}

  public async decide(input: any, _noCache?: boolean): Promise<Decision> {
    const signature: Buffer = await this.signedByDb.getMessageSignature(
      input.message,
      input.publicKey
    )

    if (!signature && !input.publicKey.equals(this.myAddress)) {
      throw new CannotDecideError(
        'We do not have a signature for this public key and message, but we do not know for certain that the message was not signed by the private key associated with the provided public key.'
      )
    }

    return this.constructDecision(signature, input.publicKey, input.message)
  }

  /**
   * Builds a Decision from the provided signature, public key, message, and outcome.
   *
   * @param signature The signature
   * @param publicKey The public key used with the signature
   * @param message The decrypted message
   * @returns the Decision
   */
  private constructDecision(
    signature: Buffer | undefined,
    publicKey: Buffer,
    message: Buffer
  ): Decision {
    return {
      outcome: !!signature,
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
}
