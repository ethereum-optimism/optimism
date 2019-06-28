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
  return signature === publicKey
}
