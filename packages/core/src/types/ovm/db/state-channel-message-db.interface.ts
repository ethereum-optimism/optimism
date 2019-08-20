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
   * @param channelID The channel ID
   * @returns The ParsedMessage, if one exits
   */
  getMostRecentValidStateChannelMessage(
    channelID: Buffer
  ): Promise<ParsedMessage>

  /**
   * Gets the ParsedMessage of the StateChannelMessage with the provided
   * ChannelID that is signed by the provided address and has the highest nonce.
   *
   * @param channelID The channel ID
   * @param address The signer's address
   * @returns The ParsedMessage, if one exits
   */
  getMostRecentMessageSignedBy(
    channelID: Buffer,
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
   * @param channelID The Channel ID in question
   * @returns True if exited, false otherwise.
   */
  isChannelExited(channelID: Buffer): Promise<boolean>

  /**
   * Marks the provided Channel ID as exited.
   *
   * @param channelID The Channel ID in question
   */
  markChannelExited(channelID: Buffer): Promise<void>

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
   * @param channelID The Channel ID in question
   * @returns True if so, false otherwise
   */
  channelIDExists(channelID: Buffer): Promise<boolean>
}
