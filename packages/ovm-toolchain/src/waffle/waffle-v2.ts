/* External Imports */
import { ethers, providers, Wallet, Contract } from 'ethers-v4'
import { defaultAccounts } from 'ethereum-waffle-v2'
import Ganache from 'ganache-core'

/* Internal Imports */
import { ganache } from '../ganache'
import {
  initCrossDomainMessengersVX,
  waitForCrossDomainMessages,
} from './waffle-vx'

const initCrossDomainMessengers = async (
  l1ToL2MessageDelay: number,
  l2ToL1MessageDelay: number,
  signer: any
): Promise<{
  l1CrossDomainMessenger: Contract
  l2CrossDomainMessenger: Contract
}> => {
  return initCrossDomainMessengersVX(
    l1ToL2MessageDelay,
    l2ToL1MessageDelay,
    ethers,
    signer
  )
}

/**
 * WaffleV2 MockProvider wrapper.
 */
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

  /**
   * Retrieves the wallet objects passed to this provider.
   * @returns List of wallet objects.
   */
  public getWallets(): Wallet[] {
    const items = this.options?.accounts ?? defaultAccounts
    return items.map((x: any) => new Wallet(x.secretKey, this))
  }

  /**
   * Sends an RPC call. Function is named "rpc" instead of "send" because
   * ethers will try to use the function if it's named "send".
   * @param method Ethereum RPC method to call.
   * @param params Params to the RPC method.
   * @returns Result of the RPC call.
   */
  public async rpc(method: string, params: any[] = []): Promise<any> {
    return new Promise<any>((resolve, reject) => {
      if (!!this._web3Provider) {
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
      } else {
        reject('web3Provider not defined')
      }
    })
  }
}

export const waffleV2 = {
  MockProvider,
  initCrossDomainMessengers,
  waitForCrossDomainMessages,
}
