/**
 * KeyManager handles storage and access of private keys.
 */
export interface KeyManager {
  /**
   * Creates an account.
   * @param password Password to lock the account with.
   * @returns the account's address.
   */
  createAccount(password?: string): Promise<string>

  /**
   * @returns the address of each stored account.
   */
  getAccounts(): Promise<string[]>

  /**
   * Unlocks the given account.
   * @param address Address of the account to unlock.
   * @param password Password for the account.
   * @returns `true` if the account is successfully unlocked.
   */
  unlockAccount(address: string, password?: string): Promise<boolean>

  /**
   * Locks the given account.
   * @param address Address of the account to lock.
   */
  lockAccount(address: string): Promise<void>

  /**
   * Signs a message with a given account.
   * @param address Address of the account to sign the message with.
   * @param message Message to sign.
   * @returns the signed message.
   */
  sign(address: string, message: string): Promise<string>
}
