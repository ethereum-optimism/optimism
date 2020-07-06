/* External Imports */
import {
  BaseQueuedPersistedProcessor,
  DB,
  EthereumListener,
} from '@eth-optimism/core-db'
import { BigNumber, getLogger, Logger } from '@eth-optimism/core-utils'

import { Block } from 'ethers/providers'

const log: Logger = getLogger('chain-data-persister')

/**
 * Base class for subscribing to block data and processing it in sequential order.
 */
export abstract class ChainDataProcessor
  extends BaseQueuedPersistedProcessor<Block>
  implements EthereumListener<Block> {
  protected constructor(db: DB, persistenceKey: string) {
    super(db, persistenceKey)
  }

  /**
   * @inheritDoc
   */
  public async handle(block: Block): Promise<void> {
    log.debug(`Received block ${block.number}.`)

    return this.add(block.number, block)
  }

  /**
   * @inheritDoc
   */
  public async onSyncCompleted(syncIdentifier?: string): Promise<void> {
    return undefined
  }

  /**
   * @inheritDoc
   */
  protected async deserializeItem(itemBuffer: Buffer): Promise<Block> {
    return JSON.parse(itemBuffer.toString('utf-8'), (key, val) => {
      if (key === 'gasLimit' || key === 'gasUsed') {
        return !!val ? new BigNumber(val, 'hex') : undefined
      }
      return val
    })
  }

  /**
   * @inheritDoc
   */
  protected async serializeItem(item: Block): Promise<Buffer> {
    return Buffer.from(
      JSON.stringify(item, (key, val) => {
        if (key === 'gasLimit' || key === 'gasUsed') {
          try {
            return val.toHexString()
          } catch (e) {
            // need to use null because undefined will omit the value.
            return null
          }
        }
        return val
      }),
      'hex'
    )
  }
}
