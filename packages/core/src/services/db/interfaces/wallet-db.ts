/* External Imports */
import { Service, OnStart } from '@nestd/core'
import { account as Account } from 'eth-lib'

/* Services */
import { DBService } from '../db.service'

/* Internal Imports */
import { EthereumAccount } from '../../../models/eth'
import { BaseDBProvider } from '../backends/base-provider'

@Service()
export class WalletDB implements OnStart {
  constructor(private readonly dbservice: DBService) {}

  /**
   * @returns the current DB instance.
   */
  public get db(): BaseDBProvider {
    const db = this.dbservice.dbs.wallet
    if (db === undefined) {
      throw new Error('WalletDB is not yet initialized.')
    }
    return db
  }

  public async onStart(): Promise<void> {
    await this.dbservice.open('wallet')
  }

  /**
   * Returns all available accounts.
   * @returns a list of account addresses.
   */
  public async getAccounts(): Promise<string[]> {
    return (await this.db.get('accounts', [])) as string[]
  }

  /**
   * Returns an account object for a given address.
   * @param address Adress of the account.
   * @returns an Ethereum account object.
   */
  public async getAccount(address: string): Promise<EthereumAccount> {
    const keystore = (await this.db.get(
      `keystore:${address}`,
      undefined
    )) as EthereumAccount
    if (keystore === undefined) {
      throw new Error('Account not found.')
    }

    return Account.fromPrivate(keystore.privateKey)
  }

  /**
   * Adds an account to the database.
   * @param account An Ethereum account object.
   */
  public async addAccount(account: EthereumAccount): Promise<void> {
    const accounts = await this.getAccounts()
    accounts.push(account.address)
    await this.db.set('accounts', accounts)
    await this.db.set(`keystore:${account.address}`, account)
  }
}
