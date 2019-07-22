/**
 * Checks that the message with the provided signature was signed by the private key
 * associated with the provided public key.
 *
 * @param signature the signed message
 * @param message the message in question
 * @param publicKey the public key to check the signature against
 * @returns true if the signature matches the message when decrypted by the publicKey
 */
export const verifySignature = (
  signature: any,
  message: any,
  publicKey: any
): boolean => {
  // TODO: Make this do actual signature checking
  return signature === message
}

/**
 * Signs the provided message with the provided key
 *
 * @param key the key with which the message should be signed
 * @param message the message to be signed
 *
 * @returns the signed message
 */
export const sign = (key: any, message: any): any => {
  // TODO: Actually sign
  return message
}

/**
 * Decrypts the provided encrypted message with the provided public key
 *
 * @param publickey the public key in question
 * @param encryptedMessage the encrypted message to decrypt
 */
export const decryptWithPublicKey = (
  publickey: any,
  encryptedMessage: any
): any => {
  // TODO: Actually decrypt
  return encryptedMessage
}
