import {
  BigNumber,
  Message,
  ParsedMessage,
  Property,
  SignedMessage,
} from '../../../types'
import {
  AndDecider,
  ForAllSuchThatDecider,
  MessageNonceLessThanDecider,
} from '../../ovm/deciders'
import {
  SignedByDecider,
  SignedByInput,
} from '../../ovm/deciders/signed-by-decider'
import { SignedByQuantifier } from '../../ovm/quantifiers/signed-by-quantifier'
import {
  deserializeBuffer,
  deserializeMessage,
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
    left: {
      decider: SignedByDecider // Asserts this message is signed by counter-party
      input: SignedByInput
    }
    leftWitness: any
    right: {
      decider: ForAllSuchThatDecider
      input: {
        quantifier: SignedByQuantifier
        quantifierParameters: any
        propertyFactory: NonceLessThanPropertyFactory
      }
    }
    rightWitness: any
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
 * @returns the resulting ParsedMessage.
 */
export const parseStateChannelSignedMessage = (
  signedMessage: SignedMessage,
  myAddress: Buffer
): ParsedMessage => {
  // TODO: Would usually decrypt message based on sender key, but that part omitted for simplicity
  const message: Message = deserializeBuffer(
    signedMessage.signedMessage,
    deserializeMessage,
    stateChannelMessageDeserializer
  )
  const signatures = {}
  signatures[signedMessage.sender.toString()] = signedMessage.signedMessage
  return {
    sender: signedMessage.sender,
    recipient: myAddress,
    message,
    signatures,
  }
}
