/* External Imports */
import { Service } from '@nestd/core'
import * as web3Utils from 'web3-utils'
import { account as Account } from 'eth-lib'

/* Services */
import { EthService } from './eth/eth.service'
import { WalletDB } from './db/interfaces/wallet-db'
import { LoggerService } from './logger.service'

/* Internal Imports */
import { EthereumAccount } from '../models/eth'

/**
 * Service used to manage the local wallet.
 */
@Service()
export class WalletService {
  private readonly name = 'wallet'

  constructor(
    private readonly logger: LoggerService,
    private readonly eth: EthService,
    private readonly walletdb: WalletDB
  ) {}

  /**
   * Returns the addresses of all accounts in this wallet.
   * @returns the list of addresses in this wallet.
   */
  public async getAccounts(): Promise<string[]> {
    return this.walletdb.getAccounts()
  }

  /**
   * @returns the keystore file for a given account.
   */
  public async getAccount(address: string): Promise<EthereumAccount> {
    return this.walletdb.getAccount(address)
  }

  /**
   * Signs a piece of arbitrary data.
   * @param address Address of the account to sign with.
   * @param data Data to sign
   * @returns a signature over the data.
   */
  public async sign(address: string, data: string): Promise<string> {
    const hash = web3Utils.sha3(data)
    const account = await this.getAccount(address)
    const sig = Account.sign(hash, account.privateKey)
    return sig.toString()
  }

  /**
   * Creates a new account.
   * @returns the account's address.
   */
  public async createAccount(): Promise<string> {
    // TODO: Support encrypted accounts.
    const account = Account.create()
    await this.walletdb.addAccount(account)
    await this.addAccountToWallet(account.address)
    this.logger.log(this.name, `Created account: ${account.address}`)
    return account.address
  }

  /**
   * Adds an account to the web3 wallet so that it can send contract
   * transactions directly. See https://bit.ly/2MPAbRd for more information.
   * @param address Address of the account to add to wallet.
   */
  public async addAccountToWallet(address: string): Promise<void> {
    const hasAccount = await this.eth.hasWalletAccount(address)
    if (hasAccount) {
      return
    }

    const account = await this.getAccount(address)
    await this.eth.addWalletAccount(account.privateKey)
  }
}
