import { Quantifier, QuantifierResult } from '../../../types/ovm'
import { SignedByDBInterface } from '../../../types/ovm/db/signed-by-db.interface'
import { deserializeMessage } from '../../serialization'
import { Message, SignedMessage } from '../../../types/serialization'

interface SignedByQuantifierParameters {
  address: string
  channelID?: string
}

/*
 * The SignedByQuantifier a collection of messages that have been signed by the provided
 */
export class SignedByQuantifier implements Quantifier {
  constructor(
    private readonly db: SignedByDBInterface,
    private readonly myAddress: string
  ) {}

  /**
   * Returns a QuantifierResult where results are an array of messages signed by the provided address.
   *
   * @param signerParams the parameters containing the signer and any other necessary info
   */
  public async getAllQuantified(
    signerParams: SignedByQuantifierParameters
  ): Promise<QuantifierResult> {
    let signedMessages: SignedMessage[] = await this.db.getAllSignedBy(
      signerParams.address
    )

    if ('channelID' in signerParams && signerParams['channelID']) {
      signedMessages = signedMessages.filter((m) => {
        try {
          const message: Message = deserializeMessage(m.serializedMessage)
          return signerParams.channelID === message.channelID
        } catch (e) {
          return false
        }
      })
    }

    return {
      results: signedMessages,
      allResultsQuantified: signerParams.address === this.myAddress,
    }
  }
}
