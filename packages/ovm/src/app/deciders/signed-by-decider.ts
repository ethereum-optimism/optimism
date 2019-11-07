import { Decider, Decision, SignedByDBInterface } from '../../types'
import { CannotDecideError } from './utils'

export interface SignedByInput {
  publicKey: string
  serializedMessage: string
}

/**
 * Decider that determines whether the provided witness is the provided message signed by
 * the private key associated with the provided public key.
 */
export class SignedByDecider implements Decider {
  constructor(
    private readonly signedByDb: SignedByDBInterface,
    private readonly myAddress: string
  ) {}

  public async decide(input: any, _noCache?: boolean): Promise<Decision> {
    const signature: string = await this.signedByDb.getMessageSignature(
      input.serializedMessage,
      input.publicKey
    )

    if (!signature && input.publicKey !== this.myAddress) {
      throw new CannotDecideError(
        'We do not have a signature for this public key and message, but we do not know for certain that the message was not signed by the private key associated with the provided public key.'
      )
    }

    return this.constructDecision(
      signature,
      input.publicKey,
      input.serializedMessage
    )
  }

  /**
   * Builds a Decision from the provided signature, public key, serializedMessage, and outcome.
   *
   * @param signature The signature
   * @param publicKey The public key used with the signature
   * @param serializedMessage The decrypted serializedMessage
   * @returns the Decision
   */
  private constructDecision(
    signature: string | undefined,
    publicKey: string,
    serializedMessage: string
  ): Decision {
    return {
      outcome: !!signature,
      justification: [
        {
          implication: {
            decider: this,
            input: {
              publicKey,
              serializedMessage,
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
