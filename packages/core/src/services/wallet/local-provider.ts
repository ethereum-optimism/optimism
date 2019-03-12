/* External Imports */
import { Service } from '@nestd/core'
import * as web3Utils from 'web3-utils'
import { account as Account } from 'eth-lib'

/* Services */
import { ETHProvider } from '../eth/eth-provider'
import { WalletDB } from '../db/interfaces/wallet-db'
import { LoggerService } from '../logger.service'

/* Internal Imports */
import { BaseWalletProvider } from './base-provider'
import { EthereumAccount } from '../../models/eth'

@Service()
export class LocalWalletProvider implements BaseWalletProvider {
  private readonly name = 'wallet'

  constructor(
    private readonly logger: LoggerService,
    private readonly eth: ETHProvider,
    private readonly walletdb: WalletDB
  ) {}

  public async getAccounts(): Promise<string[]> {
    return this.walletdb.getAccounts()
  }

  public async getAccount(address: string): Promise<EthereumAccount> {
    return this.walletdb.getAccount(address)
  }

  public async sign(address: string, data: string): Promise<string> {
    const hash = web3Utils.sha3(data)
    const account = await this.getAccount(address)
    const sig = Account.sign(hash, account.privateKey)
    return sig.toString()
  }

  public async createAccount(): Promise<string> {
    // TODO: Support encrypted accounts.
    const account = Account.create()
    await this.walletdb.addAccount(account)
    await this.addAccountToWallet(account.address)
    this.logger.log(this.name, `Created account: ${account.address}`)
    return account.address
  }

  public async addAccountToWallet(address: string): Promise<void> {
    const hasAccount = await this.eth.hasWalletAccount(address)
    if (hasAccount) {
      return
    }

    const account = await this.getAccount(address)
    await this.eth.addWalletAccount(account.privateKey)
  }
}
