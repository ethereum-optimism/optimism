/* Imports: External */
import { BaseService, LegacyMetrics } from '@eth-optimism/common-ts'
import { StaticJsonRpcProvider } from '@ethersproject/providers'
import { getChainId, sleep, toRpcHexString } from '@eth-optimism/core-utils'
import { BigNumber } from 'ethers'
import { LevelUp } from 'levelup'
import axios from 'axios'
import bfj from 'bfj'
import { Gauge, Histogram } from 'prom-client'

/* Imports: Internal */
import { handleSequencerBlock } from './handlers/transaction'
import { TransportDB } from '../../db/transport-db'
import { validators } from '../../utils'
import { L1DataTransportServiceOptions } from '../main/service'

interface L2IngestionMetrics {
  highestSyncedL2Block: Gauge<string>
  fetchBlocksRequestTime: Histogram<string>
}

const registerMetrics = ({
  client,
  registry,
}: LegacyMetrics): L2IngestionMetrics => ({
  highestSyncedL2Block: new client.Gauge({
    name: 'data_transport_layer_highest_synced_l2_block',
    help: 'Highest Synced L2 Block Number',
    registers: [registry],
  }),
  fetchBlocksRequestTime: new client.Histogram({
    name: 'data_transport_layer_fetch_blocks_time',
    help: 'Amount of time fetching remote L2 blocks takes',
    buckets: [0.1, 5, 15, 50, 100, 500],
    registers: [registry],
  }),
})

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

  private l2IngestionMetrics: L2IngestionMetrics

  private state: {
    db: TransportDB
    l2RpcProvider: StaticJsonRpcProvider
  } = {} as any

  protected async _init(): Promise<void> {
    if (this.options.legacySequencerCompatibility) {
      this.logger.info(
        'Using legacy sync, this will be quite a bit slower than normal'
      )
    }

    this.l2IngestionMetrics = registerMetrics(this.metrics)

    this.state.db = new TransportDB(this.options.db, {
      l2ChainId: this.options.l2ChainId,
    })

    this.state.l2RpcProvider =
      typeof this.options.l2RpcProvider === 'string'
        ? new StaticJsonRpcProvider({
            url: this.options.l2RpcProvider,
            user: this.options.l2RpcProviderUser,
            password: this.options.l2RpcProviderPassword,
            headers: { 'User-Agent': 'data-transport-layer' },
          })
        : this.options.l2RpcProvider
  }

  protected async ensure(): Promise<void> {
    let retries = 0
    while (true) {
      try {
        await this.state.l2RpcProvider.getNetwork()
        break
      } catch (e) {
        retries++
        this.logger.info(`Cannot connect to L2, retrying ${retries}/20`)
        if (retries >= 20) {
          this.logger.info('Cannot connect to L2, shutting down')
          await this.stop()
          process.exit()
        }
        await sleep(1000 * retries)
      }
    }
  }

  protected async checkConsistency(): Promise<void> {
    const chainId = await getChainId(this.state.l2RpcProvider)
    const shouldDoCheck = !(await this.state.db.getConsistencyCheckFlag())
    if (shouldDoCheck && chainId === 69) {
      this.logger.info('performing consistency check')
      const highestBlock =
        await this.state.db.getHighestSyncedUnconfirmedBlock()
      for (let i = 0; i < highestBlock; i++) {
        const block = await this.state.db.getUnconfirmedTransactionByIndex(i)
        if (block === null) {
          this.logger.info('resetting to null block', {
            index: i,
          })
          await this.state.db.setHighestSyncedUnconfirmedBlock(i)
          break
        }

        // Log some progress so people know what's goin on.
        if (i % 10000 === 0) {
          this.logger.info(`consistency check progress`, {
            index: i,
          })
        }
      }
      this.logger.info('consistency check complete')
      await this.state.db.putConsistencyCheckFlag(true)
    }
  }

  protected async _start(): Promise<void> {
    await this.ensure()
    await this.checkConsistency()

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
        // Also wait on edge case of no L2 transactions
        if (
          highestSyncedL2BlockNumber === targetL2Block ||
          currentL2Block === 0
        ) {
          this.logger.info(
            'All Layer 2 (Optimism) transactions are synchronized',
            {
              currentL2Block,
              targetL2Block,
            }
          )
          await sleep(this.options.pollingInterval)
          continue
        }

        this.logger.info(
          'Synchronizing unconfirmed transactions from Layer 2 (Optimism)',
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

        this.l2IngestionMetrics.highestSyncedL2Block.set(targetL2Block)

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
   *
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
      // This request returns a large response.  Parsing it into JSON inside the ethers library is
      // quite slow, and can block the event loop for upwards of multiple seconds.  When this happens,
      // incoming http requests will likely timeout and fail.
      // Instead, we will parse the incoming http stream directly with the bfj package, which yields
      // the event loop periodically so that we don't fail to serve requests.
      const req = {
        jsonrpc: '2.0',
        method: 'eth_getBlockRange',
        params: [
          toRpcHexString(startBlockNumber),
          toRpcHexString(endBlockNumber),
          true,
        ],
        id: '1',
      }

      // Retry the `eth_getBlockRange` query in case the endBlockNumber
      // is greater than the tip and `null` is returned. This gives time
      // for the sync to catch up
      let result = null
      let retry = 0
      while (result === null) {
        if (retry === 6) {
          throw new Error(
            `unable to fetch block range [${startBlockNumber},${endBlockNumber})`
          )
        }

        const end = this.l2IngestionMetrics.fetchBlocksRequestTime.startTimer()

        const resp = await axios.post(
          this.state.l2RpcProvider.connection.url,
          req,
          { responseType: 'stream' }
        )
        const respJson = await bfj.parse(resp.data, {
          yieldRate: 4096, // this yields abit more often than the default of 16384
        })

        end()

        result = respJson.result
        if (result === null) {
          retry++
          this.logger.info(
            `request for block range [${startBlockNumber},${endBlockNumber}) returned null, retry ${retry}`
          )
          await sleep(1000 * retry)
        }
      }

      blocks = result
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
