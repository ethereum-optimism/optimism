/* External Imports */
import {
  BigNumber,
  SignatureVerifier,
  DefaultSignatureVerifier,
} from '@eth-optimism/core-utils'

/* Internal Imports */
import { Message, ParsedMessage, Property, SignedMessage } from '../../../types'
import {
  AndDecider,
  ForAllSuchThatDecider,
  MessageNonceLessThanDecider,
  SignedByDecider,
  SignedByInput,
} from '../../deciders'
import { SignedByQuantifier } from '../../quantifiers'
import {
  deserializeMessage,
  deserializeMessageString,
  stateChannelMessageDeserializer,
} from '../serializers'

/*
INTERFACES FOR StateChannelExitClaim
 */
export interface NonceLessThanProperty {
  decider: MessageNonceLessThanDecider
  input: any
}

export type NonceLessThanPropertyFactory = (input: any) => NonceLessThanProperty

export interface StateChannelExitClaim extends Property {
  decider: AndDecider
  input: {
    properties: [
      {
        decider: SignedByDecider // Asserts this message is signed by counter-party
        input: SignedByInput
        witness: {}
      },
      {
        decider: ForAllSuchThatDecider
        input: {
          quantifier: SignedByQuantifier
          quantifierParameters: any
          propertyFactory: NonceLessThanPropertyFactory
        }
      }
    ]
  }
}

/*
INTERFACES FOR StateChannelMessage
 */
export interface AddressBalance {
  [address: string]: BigNumber
}

export interface StateChannelMessage {
  addressBalance: AddressBalance
}

/**
 * Parses the signed message into a ParsedMessage, if possible.
 * If not, it throws.
 *
 * @param signedMessage The signed message to parse.
 * @param myAddress The address of the caller.
 * @param signatureVerifier The [Optional] signature verifier.
 * @returns the resulting ParsedMessage.
 */
export const parseStateChannelSignedMessage = async (
  signedMessage: SignedMessage,
  myAddress: string,
  signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance()
): Promise<ParsedMessage> => {
  // TODO: Would usually decrypt message based on sender key, but that part omitted for simplicity
  const message: Message = deserializeMessageString(
    signedMessage.serializedMessage,
    deserializeMessage,
    stateChannelMessageDeserializer
  )
  const sender = await signatureVerifier.verifyMessage(
    signedMessage.serializedMessage,
    signedMessage.signature
  )

  const signatures = {}
  signatures[sender] = signedMessage.signature
  return {
    sender,
    recipient: myAddress,
    message,
    signatures,
  }
}
