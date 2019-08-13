import { Decider, Decision, ImplicationProofItem } from '../../../../types/ovm'
import { ParsedMessage } from '../../../../types/serialization'
import { BigNumber } from '../../../utils'

export interface MessageNonceLessThanInput {
  messageWithNonce: ParsedMessage
  lessThanThis: BigNumber
}

/**
 * Decider that decides true iff the input message has a nonce less than the input nonce.
 */
export class MessageNonceLessThanDecider implements Decider {
  private static _instance: MessageNonceLessThanDecider
  public static instance(): MessageNonceLessThanDecider {
    if (!MessageNonceLessThanDecider._instance) {
      MessageNonceLessThanDecider._instance = new MessageNonceLessThanDecider()
    }
    return MessageNonceLessThanDecider._instance
  }

  public async decide(
    input: MessageNonceLessThanInput,
    witness: undefined,
    noCache?: boolean
  ): Promise<Decision> {
    const justification: ImplicationProofItem[] = [
      {
        implication: {
          decider: this,
          input,
        },
        implicationWitness: witness,
      },
    ]

    return {
      outcome: input.messageWithNonce.message.nonce.lt(input.lessThanThis),
      justification,
    }
  }
}
