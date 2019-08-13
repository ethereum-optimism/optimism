import { BigNumber } from '../../number'
import { ParsedMessage } from '../../serialization'

export interface MessageDB {
  /**
   * Returns the address representing the source address of the current user's messages.
   *
   * @returns The address of the current user.
   */
  getMyAddress(): Buffer

  /**
   * Stores the provided message, updating it if one already exists.
   *
   * @param message the ParsedMessage to store
   */
  storeMessage(message: ParsedMessage): Promise<void>

  /**
   * Gets the ParsedMessage sent by the counterparty that has the same channelID and nonce
   * but different data than the one that created locally and stored, if one exists.
   *
   * @param channelId the channel ID in question
   * @param nonce the nonce in question
   * @returns The message, if there is one
   */
  getConflictingCounterpartyMessage(
    channelId: Buffer,
    nonce: BigNumber
  ): Promise<ParsedMessage>

  /**
   * Gets a specific message by the provided channel ID and nonce.
   *
   * @param channelId the channel ID in question
   * @param nonce the nonce in question
   * @returns The message, if there is one
   */
  getMessageByChannelIdAndNonce(
    channelId: Buffer,
    nonce: BigNumber
  ): Promise<ParsedMessage>

  /**
   * Gets all messages signed by the provided signer address.
   *
   * @param signer the signer address to filter by
   * @param channelId an optional channelId to filter by
   * @param nonce an optional nonce to filter by
   * @returns the list of ParsedMessages that match the provided filters
   */
  getMessagesSignedBy(
    signer: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]>

  /**
   * Gets all messages by the provided sender address.
   *
   * @param sender the sender address to filter by
   * @param channelId an optional channelId to filter by
   * @param nonce an optional nonce to filter by
   * @returns the list of ParsedMessages that match the provided filters
   */
  getMessagesBySender(
    sender: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]>

  /**
   * Gets all messages by the provided recipient address.
   *
   * @param recipient the recipient address to filter by
   * @param channelId an optional channelId to filter by
   * @param nonce an optional nonce to filter by
   * @returns the list of ParsedMessages that match the provided filters
   */
  getMessagesByRecipient(
    recipient: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]>
}
