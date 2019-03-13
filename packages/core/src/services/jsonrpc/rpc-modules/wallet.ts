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

  public async createWallet(): Promise<string> {
    return this.wallet.createAccount()
  }

  public async getAccounts(): Promise<string[]> {
    return this.wallet.getAccounts()
  }

  public async sign(address: string, data: string): Promise<string> {
    return this.wallet.sign(address, data)
  }
}
