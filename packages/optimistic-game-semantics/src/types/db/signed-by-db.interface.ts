import { MessageSubscriber } from '../message-subscriber.interface'
import { SignedMessage } from '../serialization'

export interface SignedByDBInterface extends MessageSubscriber {
  /**
   * Stores the signature for the provided signer.
   *
   * @param serializedMessage The serialized message that the signature is for
   * @param signature The signature in question
   */
  storeSignedMessage(
    serializedMessage: string,
    signature: string
  ): Promise<void>

  /**
   * Retrieves the signature for the provided message and signer public key
   * if one has been stored.
   *
   * @param serializedMessage The message of the desired signature
   * @param signerPublicKey The public key of the signer
   * @returns The signature, if one is known, for the provided message
   */
  getMessageSignature(
    serializedMessage: string,
    signerPublicKey: string
  ): Promise<string | undefined>

  /**
   * Gets all messages signed by the provided public key.
   *
   * @param publicKey The public key in question
   * @returns The serialized messages signed by the provided public key
   * TODO: we probably want this to return an object with metadata about type of message
   */
  getAllSignedBy(publicKey: string): Promise<SignedMessage[]>
}
