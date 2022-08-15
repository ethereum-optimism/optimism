import { ethers, providers } from 'ethers'
import { fetchJson, getAddress } from 'ethers/lib/utils'

export class Faucet {
  private url: string

  private provider: providers.Provider

  constructor(url: string, provider: providers.Provider) {
    this.url = url
    this.provider = provider
  }

  public async drip(
    recipient: string
  ): Promise<ethers.providers.TransactionReceipt> {
    const res = await fetchJson(
      `${this.url}/api/claim`,
      JSON.stringify({
        address: getAddress(recipient),
      })
    )

    const txHash = res.tx_hash
    return this.provider.waitForTransaction(txHash, 0, 30000)
  }
}
