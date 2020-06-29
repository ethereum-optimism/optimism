/* External Imports */
import { EthereumEvent, EthereumListener } from '@eth-optimism/core-db'

import { Event } from 'ethers'
import { JsonRpcProvider } from 'ethers/providers'

/* Internal Imports */

export class BatchItemProcessor<T> implements EthereumListener<EthereumEvent> {
  constructor(
    private readonly eventId: string,
    private readonly provider: JsonRpcProvider
  ) {}

  public async handle(t: EthereumEvent): Promise<void> {
    if (t.eventID !== this.eventId) {
      return
    }

    const tx = await this.provider.getTransaction(t.transactionHash)
  }

  public async onSyncCompleted(syncIdentifier?: string): Promise<void> {
    return undefined
  }
}
