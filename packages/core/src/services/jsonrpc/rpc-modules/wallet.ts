/* External Imports */
import { Service } from '@nestd/core'

/* Services */
import { WalletService } from '../../wallet.service'

/* Internal Imports */
import { BaseRpcModule } from './base-rpc-module'

/**
 * Subdispatcher that handles wallet-related requests.
 */
@Service()
export class WalletRpcModule extends BaseRpcModule {
  public readonly prefix = 'pg_'

  constructor(private readonly wallet: WalletService) {
    super()
  }

  /**
   * Creates a new account with a random key.
   * Stores the account on the client.
   * @returns the new account's address.
   */
  public async createAccount(): Promise<string> {
    return this.wallet.createAccount()
  }

  /**
   * @returns addresses for all accounts in the wallet.
   */
  public async getAccounts(): Promise<string[]> {
    return this.wallet.getAccounts()
  }

  /**
   * Signs a message with the given acccount.
   * @param address Address of the account to sign with.
   * @param data Message to sign.
   * @returns the signature over the given message.
   */
  public async sign(address: string, data: string): Promise<string> {
    return this.wallet.sign(address, data)
  }
}
