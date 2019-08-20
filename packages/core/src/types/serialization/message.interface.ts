import { BigNumber } from '../number'

export interface SignedMessage {
  sender: Buffer
  signedMessage: Buffer
}

export interface Message {
  channelID: Buffer
  nonce?: BigNumber
  data: {}
}

export interface ParsedMessage {
  sender: Buffer
  recipient: Buffer
  message: Message
  signatures: Signatures
}

export interface Signatures {
  [address: string]: Buffer
}
