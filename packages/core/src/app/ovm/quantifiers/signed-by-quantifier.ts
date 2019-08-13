import { Quantifier, QuantifierResult } from '../../../types/ovm'
import { MessageDB } from '../../../types/ovm/db'
import { ParsedMessage } from '../../../types/serialization'

interface SignedByQuantifierParameters {
  address: Buffer
}

/*
 * The SignedByQuantifier a collection of messages that have been signed by the provided
 */
export class SignedByQuantifier implements Quantifier {
  constructor(
    private readonly db: MessageDB,
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
    const messages: ParsedMessage[] = await this.db.getMessagesSignedBy(
      signerParams.address
    )

    return {
      results: messages,
      allResultsQuantified: signerParams.address === this.myAddress,
    }
  }
}
