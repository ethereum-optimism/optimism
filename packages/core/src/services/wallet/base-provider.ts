export interface BaseWalletProvider {
  /**
   * Returns the addresses of all accounts in this wallet.
   * @returns the list of addresses in this wallet.
   */
  getAccounts(): Promise<string[]>

  /**
   * @returns the keystore file for a given account.
   */
  getAccount(address: string): Promise<{}>

  /**
   * Signs a piece of arbitrary data.
   * @param address Address of the account to sign with.
   * @param data Data to sign
   * @returns a signature over the data.
   */
  sign(address: string, data: string): Promise<string>

  /**
   * Creates a new account.
   * @returns the account's address.
   */
  createAccount(): Promise<string>

  /**
   * Adds an account to the web3 wallet so that it can send contract
   * transactions directly. See https://bit.ly/2MPAbRd for more information.
   * @param address Address of the account to add to wallet.
   */
  addAccountToWallet(address: string): Promise<void>
}
