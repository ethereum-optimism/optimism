import { Quantifier, QuantifierResult } from '../../../types/ovm'
import { SignedByDBInterface } from '../../../types/ovm/db/signed-by-db.interface'
import { deserializeBuffer, deserializeMessage } from '../../serialization'
import { Message } from '../../../types/serialization'

interface SignedByQuantifierParameters {
  address: Buffer
  channelID?: Buffer
}

/*
 * The SignedByQuantifier a collection of messages that have been signed by the provided
 */
export class SignedByQuantifier implements Quantifier {
  constructor(
    private readonly db: SignedByDBInterface,
    private readonly myAddress: Buffer
  ) {}

  /**
   * Returns a QuantifierResult where results are an array of messages signed by the provided address.
   *
   * @param signerParams the parameters containing the signer and any other necessary info
   */
  public async getAllQuantified(
    signerParams: SignedByQuantifierParameters
  ): Promise<QuantifierResult> {
    let messages: Buffer[] = await this.db.getAllSignedBy(signerParams.address)

    if ('channelID' in signerParams && signerParams['channelID']) {
      messages = messages.filter((m) => {
        try {
          const message: Message = deserializeBuffer(m, deserializeMessage)
          return signerParams.channelID.equals(message.channelID)
        } catch (e) {
          return false
        }
      })
    }

    return {
      results: messages,
      allResultsQuantified: signerParams.address === this.myAddress,
    }
  }
}
