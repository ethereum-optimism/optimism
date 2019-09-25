import { Message, SignedMessage } from '../serialization'

/**
 * Interface to allow generic Message subscription and processing.
 */
export interface MessageSubscriber {
  /**
   * Handles the provided message however its logic specifies.
   *
   * @param serializedMessage The serialized message to handle.
   * @param signature The signature if there is one
   */
  handleMessage(serializedMessage: string, signature?: string): Promise<void>
}
