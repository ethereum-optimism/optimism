/* External Imports */
import { Service } from '@nestd/core'

/* Services */
import { BaseWalletProvider } from '../../wallet/base-provider'

/* Internal Imports */
import { BaseSubdispatcher } from './base-subdispatcher'

/**
 * Subdispatcher that handles wallet-related requests.
 */
@Service()
export class WalletSubdispatcher extends BaseSubdispatcher {
  public readonly prefix = 'pg_'

  constructor(private readonly wallet: BaseWalletProvider) {
    super()
  }

  get methods(): { [key: string]: (...args: any) => any } {
    const wallet = this.wallet

    return {
      /* Wallet */
      createAccount: wallet.createAccount.bind(wallet),
      getAccounts: wallet.getAccounts.bind(wallet),
      sign: wallet.sign.bind(wallet),
    }
  }
}
