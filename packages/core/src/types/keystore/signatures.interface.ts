export interface SignatureProvider {
  /**
   * Signs the provided message with the private key associated with the provided address.
   *
   * @param address The address
   * @param message The message
   * @returns the signature
   */
  sign(address: string, message: string): Promise<string>
}

export interface SignatureVerifier {
  /**
   * Gets the address that signed the provided message, resulting in the
   * provided signature.
   *
   * @param message The message that was signed.
   * @param signature The signature of the message that was signed.
   * @returns The signer's address.
   */
  verifyMessage(message: string, signature: string): string
}
