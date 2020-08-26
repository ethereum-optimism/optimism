/* External Imports */
import {
  BaseQueuedPersistedProcessor,
  EthereumListener,
  RDB,
  SequentialProcessingDataService,
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
  protected constructor(
    processingDataService: SequentialProcessingDataService,
    persistenceKey: string,
    startIndex: number
  ) {
    super(processingDataService, persistenceKey, startIndex)
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
  protected async deserializeItem(itemString: string): Promise<Block> {
    if (!itemString || itemString.length === 0) {
      log.error(`Deserialized empty block ${itemString}. Returning undefined.`)
      return undefined
    }

    return JSON.parse(itemString, (key, val) => {
      if (key === 'gasLimit' || key === 'gasUsed') {
        return !!val ? new BigNumber(val, 'hex') : undefined
      }
      return val
    })
  }

  /**
   * @inheritDoc
   */
  protected async serializeItem(item: Block): Promise<string> {
    return JSON.stringify(item, (key, val) => {
      if (key === 'gasLimit' || key === 'gasUsed') {
        try {
          return val.toHexString()
        } catch (e) {
          log.debug(`Error converting key ${key} to hex. Val: ${val}.`)
          // need to use null because undefined will omit the value.
          return null
        }
      }
      return val
    })
  }
}
