/* External Imports */
import { getLogger, Logger, numberToHexString } from '@eth-optimism/core-utils'

import { JsonRpcProvider } from 'ethers/providers'
import { Wallet } from 'ethers'

/* Internal Imports */
import { BlockBatches, L2NodeService } from '../types'

const log: Logger = getLogger('block-batch-submitter')

export class DefaultL2NodeService implements L2NodeService {
  // params: [blockBatchesJSONString, signedBlockBatchesJSONString]
  // -- note all numbers are replaces with hex strings when serialized
  public static readonly sendBlockBatchesMethod: string = 'eth_sendBlockBatches'

  private readonly l2Provider: JsonRpcProvider

  constructor(private readonly l2Wallet: Wallet) {
    this.l2Provider = l2Wallet.provider as JsonRpcProvider
  }

  /**
   * @inheritDoc
   */
  public async sendBlockBatches(blockBatches: BlockBatches): Promise<void> {
    if (!blockBatches) {
      const msg = `Received undefined Block Batch!.`
      log.error(msg)
      throw msg
    }

    if (!blockBatches.batches || !blockBatches.batches.length) {
      log.error(`Received empty block batch: ${JSON.stringify(blockBatches)}`)
      return
    }

    const payload = JSON.stringify(blockBatches, (k, v) => {
      if (typeof v === 'number') {
        return v >= 0 ? numberToHexString(v) : undefined
      }
      return v
    })

    const signedPayload: string = await this.l2Wallet.signMessage(payload)
    await this.l2Provider.send(DefaultL2NodeService.sendBlockBatchesMethod, [
      payload,
      signedPayload,
    ])
  }
}
