/* External Imports */
import { HashAlgorithm } from '@eth-optimism/core-utils'

/* Internal Imports */
import { MessageSubscriber } from '../message-subscriber.interface'

export interface HashPreimageDBInterface extends MessageSubscriber {
  /**
   * Calculates and stores the hash and provided preimage using the provided
   * HashAlgorithm for future lookup.
   *
   * @param preimage The preimage to store
   * @param hashAlgorithm The HashAlgorithm in question
   */
  storePreimage(preimage: string, hashAlgorithm: HashAlgorithm): Promise<void>

  /**
   * Retrieves the preimage for the provided hash, using the provided HashAlgorithm,
   * if one has been stored.
   *
   * @param hash The hash in question
   * @param hashAlgorithm The algorithm used
   * @returns The preimage, if one is known, for the provided hash
   */
  getPreimage(
    hash: string,
    hashAlgorithm: HashAlgorithm
  ): Promise<string | undefined>
}
