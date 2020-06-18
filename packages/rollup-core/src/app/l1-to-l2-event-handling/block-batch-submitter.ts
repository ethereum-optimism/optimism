/* External Imports */
import { getLogger, Logger, numberToHexString } from '@eth-optimism/core-utils'

import { JsonRpcProvider } from 'ethers/providers'
import { Wallet } from 'ethers'

/* Internal Imports */
import { BlockBatches, BlockBatchListener } from '../../types'

const log: Logger = getLogger('block-batch-submitter')

export class BlockBatchSubmitter implements BlockBatchListener {
  // params: [timestampHex, batchesArrayJSON, signedBatchesArrayJSON]
  public static readonly sendBlockBatchesMethod: string = 'eth_sendBlockBatches'

  private readonly l2Provider: JsonRpcProvider

  constructor(private readonly l2Wallet: Wallet) {
    this.l2Provider = l2Wallet.provider as JsonRpcProvider
  }

  /**
   * @inheritDoc
   */
  public async handleBlockBatches(blockBatches: BlockBatches): Promise<void> {
    if (!blockBatches) {
      const msg = `Received undefined Block Batch!.`
      log.error(msg)
      throw msg
    }

    if (!blockBatches.batches || !blockBatches.batches.length) {
      log.debug(`Moving past empty block ${blockBatches.blockNumber}.`)
      return
    }

    const payload = JSON.stringify(blockBatches, (k, v) => {
      if (typeof v === 'number') {
        return v >= 0 ? numberToHexString(v) : undefined
      }
      return v
    })

    const signedPayload: string = await this.l2Wallet.signMessage(payload)
    await this.l2Provider.send(BlockBatchSubmitter.sendBlockBatchesMethod, [
      payload,
      signedPayload,
    ])
  }
}
