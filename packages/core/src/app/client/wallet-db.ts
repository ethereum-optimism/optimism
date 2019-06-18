/* Internal Imports */
import { WalletDB, Keystore, KeyValueStore } from '../../interfaces'

export class DefaultWalletDB implements WalletDB {
  /**
   * Initializes the database wrapper.
   * @param db Database instance to insert data into.
   */
  constructor(private db: KeyValueStore) {}

  /**
   * Adds a keystore file to the database.
   * @param keystore Keystore file to add.
   * @returns a Promise that resolves once the keystore has been inserted.
   */
  public async putKeystore(keystore: Keystore): Promise<void> {
    const key = Buffer.from(keystore.address)
    const value = Buffer.from(JSON.stringify(keystore))

    await this.db.put(key, value)
  }

  /**
   * Queries a keystore file from the database.
   * @param address Address to query a keystore for.
   * @returns the keystore file for the given address.
   */
  public async getKeystore(address: string): Promise<Keystore> {
    const key = Buffer.from(address)
    const value = await this.db.get(key)

    if (value === null) {
      throw new Error('Keystore file does not exist.')
    }

    return JSON.parse(value.toString())
  }

  /**
   * Checks if the database has a specific keystore file.
   * @param address Address to find a keystore file for.
   * @returns `true` if the database has the keystore, `false` otherwise.
   */
  public async hasKeystore(address: string): Promise<boolean> {
    try {
      await this.getKeystore(address)
    } catch (err) {
      if (err.message.includes('Keystore file does not exist.')) {
        return false
      } else {
        throw err
      }
    }

    return true
  }

  /**
   * Lists all addresses that the database has keystore files for.
   * @returns all addresses with stored keystores.
   */
  public async listAccounts(): Promise<string[]> {
    const keys = await this.db.iterator().keys()
    return keys.map((key) => key.toString())
  }
}
