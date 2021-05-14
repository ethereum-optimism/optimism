/* Imports: External */
import { BaseService } from '@eth-optimism/common-ts'
import { JsonRpcProvider } from '@ethersproject/providers'
import { BigNumber } from 'ethers'
import { LevelUp } from 'levelup'

/* Imports: Internal */
import { TransportDB } from '../../db/transport-db'
import { sleep, toRpcHexString, validators } from '../../utils'
import { L1DataTransportServiceOptions } from '../main/service'
import { handleSequencerBlock } from './handlers/transaction'

export interface L2IngestionServiceOptions
  extends L1DataTransportServiceOptions {
  db: LevelUp
}

const optionSettings = {
  db: {
    validate: validators.isLevelUP,
  },
  l2RpcProvider: {
    validate: (val: any) => {
      return validators.isUrl(val) || validators.isJsonRpcProvider(val)
    },
  },
  l2ChainId: {
    validate: validators.isInteger,
  },
  pollingInterval: {
    default: 5000,
    validate: validators.isInteger,
  },
  transactionsPerPollingInterval: {
    default: 1000,
    validate: validators.isInteger,
  },
  dangerouslyCatchAllErrors: {
    default: false,
    validate: validators.isBoolean,
  },
  legacySequencerCompatibility: {
    default: false,
    validate: validators.isBoolean,
  },
}

export class L2IngestionService extends BaseService<L2IngestionServiceOptions> {
  constructor(options: L2IngestionServiceOptions) {
    super('L2_Ingestion_Service', options, optionSettings)
  }

  private state: {
    db: TransportDB
    l2RpcProvider: JsonRpcProvider
  } = {} as any

  protected async _init(): Promise<void> {
    if (this.options.legacySequencerCompatibility) {
      this.logger.info(
        'Using legacy sync, this will be quite a bit slower than normal'
      )
    }

    this.state.db = new TransportDB(this.options.db)

    this.state.l2RpcProvider =
      typeof this.options.l2RpcProvider === 'string'
        ? new JsonRpcProvider(this.options.l2RpcProvider)
        : this.options.l2RpcProvider
  }

  protected async _start(): Promise<void> {
    while (this.running) {
      try {
        const highestSyncedL2BlockNumber =
          (await this.state.db.getHighestSyncedUnconfirmedBlock()) || 1

        const currentL2Block = await this.state.l2RpcProvider.getBlockNumber()

        // Make sure we don't exceed the tip.
        const targetL2Block = Math.min(
          highestSyncedL2BlockNumber +
            this.options.transactionsPerPollingInterval,
          currentL2Block
        )

        // We're already at the head, so no point in attempting to sync.
        if (highestSyncedL2BlockNumber === targetL2Block) {
          await sleep(this.options.pollingInterval)
          continue
        }

        this.logger.info(
          'Synchronizing unconfirmed transactions from Layer 2 (Optimistic Ethereum)',
          {
            fromBlock: highestSyncedL2BlockNumber,
            toBlock: targetL2Block,
          }
        )

        // Synchronize by requesting blocks from the sequencer. Sync from L1 takes precedence.
        await this._syncSequencerBlocks(
          highestSyncedL2BlockNumber,
          targetL2Block
        )

        await this.state.db.setHighestSyncedUnconfirmedBlock(targetL2Block)

        if (
          currentL2Block - highestSyncedL2BlockNumber <
          this.options.transactionsPerPollingInterval
        ) {
          await sleep(this.options.pollingInterval)
        }
      } catch (err) {
        if (!this.running || this.options.dangerouslyCatchAllErrors) {
          this.logger.error('Caught an unhandled error', {
            message: err.toString(),
            stack: err.stack,
            code: err.code,
          })
          await sleep(this.options.pollingInterval)
        } else {
          throw err
        }
      }
    }
  }

  /**
   * Synchronizes unconfirmed transactions from a range of sequencer blocks.
   * @param startBlockNumber Block to start querying from.
   * @param endBlockNumber Block to query to.
   */
  private async _syncSequencerBlocks(
    startBlockNumber: number,
    endBlockNumber: number
  ): Promise<void> {
    if (startBlockNumber > endBlockNumber) {
      this.logger.warn(
        'Cannot query with start block number larger than end block number',
        {
          startBlockNumber,
          endBlockNumber,
        }
      )
      return
    }

    let blocks: any = []
    if (this.options.legacySequencerCompatibility) {
      const blockPromises = []
      for (let i = startBlockNumber; i <= endBlockNumber; i++) {
        blockPromises.push(
          this.state.l2RpcProvider.send('eth_getBlockByNumber', [
            toRpcHexString(i),
            true,
          ])
        )
      }

      // Just making sure that the blocks will come back in increasing order.
      blocks = (await Promise.all(blockPromises)).sort((a, b) => {
        return (
          BigNumber.from(a.number).toNumber() -
          BigNumber.from(b.number).toNumber()
        )
      })
    } else {
      blocks = await this.state.l2RpcProvider.send('eth_getBlockRange', [
        toRpcHexString(startBlockNumber),
        toRpcHexString(endBlockNumber),
        true,
      ])
    }

    for (const block of blocks) {
      const entry = await handleSequencerBlock.parseBlock(
        block,
        this.options.l2ChainId
      )
      await handleSequencerBlock.storeBlock(entry, this.state.db)
    }
  }
}
