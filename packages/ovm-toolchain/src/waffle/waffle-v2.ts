import { providers, Wallet } from 'ethers-v4'
import { defaultAccounts } from 'ethereum-waffle-v2'
import Ganache from 'ganache-core'
import { ganache } from '../ganache'

export class MockProvider extends providers.Web3Provider {
  constructor(private options?: Ganache.IProviderOptions) {
    super(
      ganache.provider({
        gasPrice: 0,
        accounts: defaultAccounts,
        ...options,
      }) as any
    )
  }

  public getWallets() {
    const items = this.options?.accounts ?? defaultAccounts
    return items.map((x: any) => new Wallet(x.secretKey, this))
  }

  public async sendRpc(method: string, params: any[] = []): Promise<any> {
    return new Promise<any>((resolve, reject) => {
      this._web3Provider.sendAsync(
        {
          jsonrpc: '2.0',
          method,
          params,
        },
        (err: any, res: any) => {
          if (err) {
            reject(err)
          } else {
            resolve(res.result)
          }
        }
      )
    })
  }
}

export const waffleV2 = {
  MockProvider,
}
