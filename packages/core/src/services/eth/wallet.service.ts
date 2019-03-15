/* External Imports */
import { Service } from '@nestd/core'
import { sha3 } from 'web3-utils'
import { account as Account } from 'eth-lib'
import Web3 from 'web3'

/* Services */
import { Web3Service } from './web3.service'
import { WalletDB } from '../db/interfaces/wallet-db'
import { LoggerService, SyncLogger } from '../logging'

/* Internal Imports */
import { EthereumAccount, isAccount } from '../../models/eth'
import { isString } from 'util'

/**
 * Service used to manage the local wallet.
 */
@Service()
export class WalletService {
  private readonly logger = new SyncLogger('wallet', this.logs)

  constructor(
    private readonly logs: LoggerService,
    private readonly web3Service: Web3Service,
    private readonly walletdb: WalletDB
  ) {}

  /**
   * @returns the current web3 instance.
   */
  get web3(): Web3 {
    return this.web3Service.web3
  }

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
    const hash = sha3(data)
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
    this.logger.log(`Created account: ${account.address}`)
    return account.address
  }

  /**
   * Adds an account to the web3 wallet so that it can send contract
   * transactions directly. See https://bit.ly/2MPAbRd for more information.
   * @param address Address of the account to add to wallet.
   */
  public async addAccountToWallet(address: string): Promise<void> {
    const hasAccount = await this.hasWalletAccount(address)
    if (hasAccount) {
      return
    }

    const account = await this.getAccount(address)
    await this.addWalletAccount(account.privateKey)
  }

  /**
   * @returns the list of address in the user's wallet.
   */
  public async getWalletAccounts(): Promise<string[]> {
    const wallet = this.web3.eth.accounts.wallet
    const keys = Object.keys(wallet)
    return keys.filter((key) => {
      return this.web3.utils.isAddress(key)
    })
  }

  /**
   * Returns the account object for a given account.
   * @param address Address of the account.
   * @returns the account object.
   */
  public async getWalletAccount(address: string): Promise<EthereumAccount> {
    const wallet = this.web3.eth.accounts.wallet
    for (const key of Object.keys(wallet)) {
      const value = wallet[key]
      if (key === address && !isString(value) && isAccount(value)) {
        return value as EthereumAccount
      }
    }

    throw new Error('Account not found.')
  }

  /**
   * Checks if the wallet has the given account.
   * @param address Address to check.
   * @returns `true` if the wallet has account, `false` otherwise.
   */
  private async hasWalletAccount(address: string): Promise<boolean> {
    const accounts = await this.getWalletAccounts()
    return accounts.includes(address)
  }

  /**
   * Adds an account to the user's wallet.
   * @param privateKey the account's private key.
   */
  private async addWalletAccount(privateKey: string): Promise<void> {
    await this.web3.eth.accounts.wallet.add(privateKey)
  }
}
