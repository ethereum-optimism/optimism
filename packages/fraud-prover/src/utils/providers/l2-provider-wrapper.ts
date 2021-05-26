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

/*
{"level":30,"time":1621976500683,"l1StateRoot":"0xbad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1","msg":"L1 State Root"}
{"level":30,"time":1621976500683,"l2StateRoot":"0x701a046ebe69e7a2745a7aeccaa3c7f3d1148cca6c8cbca445736b27f06326ca","msg":"L2 State Root"}
*/

  public async getStateDiffProof(index: number): Promise<StateDiffProof> {
    
    let proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(0),
    ])

    console.log("proof 0:",proof)

    proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(1),
    ])

    console.log("proof 1:",proof)

    proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(2),
    ])

    console.log("proof 2:",proof)

    proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(3),
    ])

    console.log("proof 3:",proof)

    proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(4),
    ])
    console.log("proof 4:",proof)

    proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(5),
    ])
    console.log("proof 5:",proof)

    proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(6),
    ])
    console.log("proof 6:",proof)

    proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(7),
    ])
    console.log("proof 7:",proof)

    proof = await this.provider.send('eth_getStateDiffProof', [
      toUnpaddedHexString(index),
    ])

    console.log("If this is empty, there is a major problem")
    console.log("proof:",proof)

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
