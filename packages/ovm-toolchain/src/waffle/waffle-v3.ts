/* External Imports */
import { providers, Wallet } from 'ethers'
import { defaultAccounts } from 'ethereum-waffle'
import Ganache from 'ganache-core'

/* Internal Imports */
import { ganache } from '../ganache'

interface MockProviderOptions {
  ganacheOptions: Ganache.IProviderOptions
}

/**
 * WaffleV3 MockProvider wrapper.
 */
export class MockProvider extends providers.Web3Provider {
  constructor(private options?: MockProviderOptions) {
    super(
      ganache.provider({
        gasPrice: 0,
        accounts: defaultAccounts,
        ...options?.ganacheOptions,
      }) as any
    )
  }

  /**
   * Retrieves the wallet objects passed to this provider.
   * @returns List of wallet objects.
   */
  public getWallets(): Wallet[] {
    const items = this.options?.ganacheOptions.accounts ?? defaultAccounts
    return items.map((x: any) => new Wallet(x.secretKey, this))
  }
}

export const waffleV3 = {
  MockProvider,
}

export const waffle = {
  MockProvider,
}
