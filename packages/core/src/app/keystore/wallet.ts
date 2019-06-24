/* External Imports */
import { ethers } from 'ethers'

/* Internal Imports */
import { Wallet, WalletDB } from '../../types'

/**
 * Simple Wallet implementation.
 */
export class DefaultWallet implements Wallet {
  private unlocked: Record<string, ethers.Wallet> = {}

  /**
   * Initializes the wallet.
   * @param walletdb DB wrapper to get values from.
   */

  constructor(private walletdb: WalletDB) {}

  /**
   * @returns the list of account addresses stored in the wallet.
   */
  public async listAccounts(): Promise<string[]> {
    return this.walletdb.listAccounts()
  }

  /**
   * Creates a new account.
   * @param password Password for the account.
   * @returns the account address.
   */
  public async createAccount(password: string): Promise<string> {
    const wallet = ethers.Wallet.createRandom()
    const keystore = await wallet.encrypt(password)
    await this.walletdb.putKeystore(JSON.parse(keystore))
    return wallet.address
  }

  /**
   * Unlocks an account.
   * @param address Account to unlock.
   * @param password Password for the account.
   */
  public async unlockAccount(address: string, password: string): Promise<void> {
    if (!(await this.walletdb.hasKeystore(address))) {
      throw new Error('Account does not exist.')
    }

    const keystore = await this.walletdb.getKeystore(address)

    let wallet
    try {
      wallet = await ethers.Wallet.fromEncryptedJson(
        JSON.stringify(keystore),
        password
      )
    } catch (err) {
      // TODO: Figure out how to handle other decryption errors.
      throw new Error('Invalid account password.')
    }

    // TODO: Is there a more secure way to store unlocked accounts?
    this.unlocked[address] = wallet
  }

  /**
   * Locks an account.
   * @param address Account to lock.
   */
  public async lockAccount(address: string): Promise<void> {
    delete this.unlocked[address]
  }

  /**
   * Signs a message.
   * @param address Address to sign the message from.
   * @param message Message to sign.
   * @returns the signature over the message.
   */
  public async sign(address: string, message: string): Promise<string> {
    if (!(await this.walletdb.hasKeystore(address))) {
      throw new Error('Account does not exist.')
    }

    if (!(address in this.unlocked)) {
      throw new Error('Account is not unlocked.')
    }

    return this.unlocked[address].signMessage(message)
  }
}
