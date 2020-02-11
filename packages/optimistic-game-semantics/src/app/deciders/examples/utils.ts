/* External Imports */
import { objectsEqual } from '@eth-optimism/core-utils/build'

/* Internal Imports */
import { ParsedMessage } from '../../../types/serialization'

export class Utils {
  /**
   * Determines whether or not the provided ParsedMessages conflict.
   * Conflicting messages have the same channelID and nonce but different data.
   *
   * @param message The first message
   * @param other The second message
   * @returns True if they conflict, false otherwise
   */
  public static stateChannelMessagesConflict(
    message: ParsedMessage,
    other: ParsedMessage
  ): boolean {
    return (
      !!message &&
      !!other &&
      message.message.channelID === other.message.channelID &&
      message.message.nonce.equals(other.message.nonce) &&
      (message.sender === other.sender || message.sender === other.recipient) &&
      (message.recipient === other.recipient ||
        message.recipient === other.sender) &&
      !objectsEqual(message.message.data, other.message.data)
    )
  }
}
