import { MessageDB } from './message-db.interface'
import { ParsedMessage } from '../../serialization'
import { SignedByDBInterface } from './signed-by-db.interface'

/**
 * The MessageDB interface specific to StateChannelMessages, offering various
 * StateChannelMessage-specific CRUD operations.
 */
export interface StateChannelMessageDBInterface
  extends MessageDB,
    SignedByDBInterface {
  /**
   * Gets the ParsedMessage of the StateChannelMessage with the provided
   * ChannelID that is signed by both parties and has the highest nonce.
   *
   * @param channelId The channel ID
   * @returns The ParsedMessage, if one exits
   */
  getMostRecentValidStateChannelMessage(
    channelId: Buffer
  ): Promise<ParsedMessage>

  /**
   * Gets the ParsedMessage of the StateChannelMessage with the provided
   * ChannelID that is signed by the provided address and has the highest nonce.
   *
   * @param channelId The channel ID
   * @param address The signer's address
   * @returns The ParsedMessage, if one exits
   */
  getMostRecentMessageSignedBy(
    channelId: Buffer,
    address: Buffer
  ): Promise<ParsedMessage>

  /**
   * Determines if the provided message conflicts with any other messages.
   * Messages conflict if they are signed by different parties, have the
   * same nonce, but have different data.
   *
   * @param message The message for which we're searching for conflicts.
   * @returns The message it conflicts with, if one exists, else undefined
   */
  conflictsWithAnotherMessage(message: ParsedMessage): Promise<ParsedMessage>

  /**
   * Determines whether the channel with the provided ChannelID has been
   * exited or has an active exit attempt by either party.
   *
   * @param channelId The Channel ID in question
   * @returns True if exited, false otherwise.
   */
  isChannelExited(channelId: Buffer): Promise<boolean>

  /**
   * Marks the provided Channel ID as exited.
   *
   * @param channelId The Channel ID in question
   */
  markChannelExited(channelId: Buffer): Promise<void>

  /**
   * Gets the ChannelID associated with the provided counterparty address.
   *
   * @param address The address in question
   * @returns The resulting ChannelID, if one exists
   */
  getChannelForCounterparty(address: Buffer): Promise<Buffer>

  /**
   * Determines whether or not the provided Channel ID represents a
   * State Channel that we are a party of.
   *
   * @param channelId The Channel ID in question
   * @returns True if so, false otherwise
   */
  channelIdExists(channelId: Buffer): Promise<boolean>
}
