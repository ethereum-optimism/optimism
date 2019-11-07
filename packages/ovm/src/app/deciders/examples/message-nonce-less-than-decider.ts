/* External Imports */
import { BigNumber } from '@pigi/core-utils'

/* Internal Imports */
import { Message } from '../../../types/serialization'
import { Decider, Decision, ImplicationProofItem } from '../../../types'

export interface MessageNonceLessThanInput {
  messageWithNonce: Message
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
    noCache?: boolean
  ): Promise<Decision> {
    const justification: ImplicationProofItem[] = [
      {
        implication: {
          decider: this,
          input,
        },
        implicationWitness: undefined,
      },
    ]

    return {
      outcome: input.messageWithNonce.nonce.lt(input.lessThanThis),
      justification,
    }
  }
}
