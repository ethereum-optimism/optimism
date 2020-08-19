import { providers, Wallet } from 'ethers-v5'
import { defaultAccounts } from 'ethereum-waffle-v3'
import Ganache from 'ganache-core'
import { ganache } from '../ganache'

interface MockProviderOptions {
  ganacheOptions: Ganache.IProviderOptions
}

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

  public getWallets() {
    const items = this.options?.ganacheOptions.accounts ?? defaultAccounts
    return items.map((x: any) => new Wallet(x.secretKey, this))
  }
}

export const waffleV3 = {
  MockProvider,
}
