import { ParsedMessage } from '../../../../types/serialization'
import { objectsEqual } from '../../../utils'

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
      message.message.channelID.equals(other.message.channelID) &&
      message.message.nonce.equals(other.message.nonce) &&
      (message.sender.equals(other.sender) ||
        message.sender.equals(other.recipient)) &&
      (message.recipient.equals(other.recipient) ||
        message.recipient.equals(other.sender)) &&
      !objectsEqual(message.message.data, other.message.data)
    )
  }
}
