//import { JsonRpcProvider } from '@ethersproject/providers'

import { providers } from 'ethers'
import { StateDiffProof } from '../../types'
import { toUnpaddedHexString } from '../hex-utils'

export class L2ProviderWrapper {
  constructor(public provider: providers.JsonRpcProvider) {}

  public async getStateRoot(index: number): Promise<string> {
    const block = await this.provider.send('eth_getBlockByNumber', [
      toUnpaddedHexString(index),
      false,
    ])
    return block.stateRoot
  }

  public async getTransaction(index: number): Promise<string> {
    const transaction = await this.provider.send(
      'eth_getTransactionByBlockNumberAndIndex',
      [toUnpaddedHexString(index), '0x0']
    )

    return transaction.input
  }

  public async getProof(
    index: number,
    address: string,
    slots: string[] = []
  ): Promise<any> {
    return this.provider.send('eth_getProof', [
      address,
      slots,
      toUnpaddedHexString(index),
    ])
  }

  public async getStateDiffProof(index: number): Promise<StateDiffProof> {
    const proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(index),
    ])

    return {
      header: proof.header,
      accountStateProofs: proof.accounts,
    }
  }

  public async getRollupInfo(): Promise<any> {
    return this.provider.send('rollup_getInfo', [])
  }

  public async getAddressManagerAddress(): Promise<string> {
    const rollupInfo = await this.getRollupInfo()
    return rollupInfo.addresses.addressResolver
  }
}
