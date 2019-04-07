import { sha3 } from 'web3-utils'
import { account as accountlib } from 'eth-lib'

import { KeyManager, KeyValueStore, Account } from '../../../interfaces'
import { BaseKey } from '../../common'

const addressKey = new BaseKey('a', ['hash160'])

/**
 * Simple key manager that opens a database for keys.
 */
export class DefaultKeyManager implements KeyManager {
  constructor(private db: KeyValueStore) {}

  /**
   * Creates an account and inserts it into the database.
   * @param [password] Password used to encrypt the account.
   * @returns the new account's address.
   */
  public async createAccount(password?: string): Promise<string> {
    const account = accountlib.create()
    await this.db.put(
      addressKey.encode([account.address]),
      Buffer.from(JSON.stringify(account), 'utf8')
    )
    return account.address
  }

  /**
   * @returns addresses of all stored accounts.
   */
  public async getAccounts(): Promise<string[]> {
    const iterator = this.db.iterator()
    const accounts = (await iterator.keys()).map((key) => {
      return key.toString('hex')
    })
    return accounts
  }

  /**
   * Unlocks an account.
   * @param address Address of the account to unlock.
   * @param password Password to unlock the account.
   * @returns `true` if the account is unlocked, `false` otherwise.
   */
  public async unlockAccount(
    address: string,
    password: string
  ): Promise<boolean> {
    // TODO: Add support for encrypted accounts.
    return true
  }

  /**
   * Locks an account.
   * @param address Address of the account to lock.
   */
  public async lockAccount(address: string): Promise<void> {
    // TODO: Add support for encrypted accounts.
  }

  /**
   * Signs a message with an account.
   * Account must be unlocked already.
   * @param address Address of the account to sign with.
   * @param message Message to sign.
   * @returns the signature over the message.
   */
  public async sign(address: string, message: string): Promise<string> {
    const account = await this.db.get(addressKey.encode([address]))
    const parsed: Account = JSON.parse(account.toString('utf8'))
    const messageHash = sha3(message)
    return accountlib.sign(messageHash, parsed.privateKey)
  }
}
