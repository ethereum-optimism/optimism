import { Message, SignedMessage } from '../serialization'

/**
 * Interface to allow generic Message subscription and processing.
 */
export interface MessageSubscriber {
  /**
   * Handles the provided message however its logic specifies.
   *
   * @param message The decrypted Message to handle
   * @param signedMessage The SignedMessage in the event the signature is relevant
   */
  handleMessage(message: Message, signedMessage?: SignedMessage): Promise<void>
}
