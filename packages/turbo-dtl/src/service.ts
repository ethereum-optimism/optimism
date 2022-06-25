/* Imports: External */
import { ethers } from 'ethers'
import {
  BaseServiceV2,
  ExpressRouter,
  Gauge,
  validators,
} from '@eth-optimism/common-ts'
import { getContractInterface } from '@eth-optimism/contracts'
import { sleep } from '@eth-optimism/core-utils'
import level from 'level'

import {
  parseTransactionEnqueued,
  parseTransactionBatchAppended,
  EventParsingFunction,
} from './parsing'
import { ErrEntryInconsistency } from './errors'
import { BatchTransactionEntry, SimpleDB, TransportDB } from './db'
import { getRangeEnd, range } from './helpers'

type DTLOptions = {
  db: string
  l1RpcProvider: ethers.providers.StaticJsonRpcProvider
  l2ChainId: number
  l1StartHeight: number
  confirmations: number
  blocksPerLogQuery: number
  addressManager: string
}

type DTLMetrics = {
  highestSyncedL1Block: Gauge
  highestSyncedL2Block: Gauge
  highestKnownL2Block: Gauge
  unexpectedRetryableErrors: Gauge
}

type DTLState = {
  db: TransportDB
  highestKnownL1Block: number
  CanonicalTransactionChain: ethers.Contract
}

export class DTLService extends BaseServiceV2<
  DTLOptions,
  DTLMetrics,
  DTLState
> {
  constructor(options?: Partial<DTLOptions>) {
    super({
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      version: require('../package.json').version,
      name: 'data-transport-layer',
      options,
      api: '/',
      port: 7878,
      optionsSpec: {
        db: {
          validator: validators.str,
          desc: 'path to database folder',
        },
        l1RpcProvider: {
          validator: validators.ethersStaticJsonRpcProvider,
          desc: 'provider for interacting with L1',
          secret: true,
        },
        l2ChainId: {
          validator: validators.num,
          desc: 'chain ID for the L2 chain',
        },
        l1StartHeight: {
          validator: validators.num,
          desc: 'L1 block height where the L2 chain starts',
        },
        confirmations: {
          validator: validators.num,
          desc: 'number of confirmations when syncing from L1',
        },
        blocksPerLogQuery: {
          validator: validators.num,
          desc: 'size of the range of the log query in block',
          default: 2000,
        },
        addressManager: {
          validator: validators.str,
          desc: 'address of the AddressManager contract on L1',
        },
      },
      metricsSpec: {
        highestSyncedL1Block: {
          type: Gauge,
          desc: 'highest synced L1 block number',
        },
        highestSyncedL2Block: {
          type: Gauge,
          desc: 'highest synced L2 block number',
        },
        highestKnownL2Block: {
          type: Gauge,
          desc: 'highest known L2 block number',
        },
        unexpectedRetryableErrors: {
          type: Gauge,
          desc: 'count of errors within the retryable function',
          labels: ['name'],
        },
      },
    })
  }

  protected async routes(router: ExpressRouter): Promise<void> {
    router.get('/eth/syncing', async (req, res) => {
      const highestSyncedL2Block = await this.state.db.getHighestSyncedL2Block()
      const highestKnownL2Block = await this.state.db.getHighestKnownL2Block()
      if (highestSyncedL2Block === null || highestKnownL2Block === null) {
        return res.json({
          syncing: true,
          currentTransactionIndex: 0,
          highestKnownTransactionIndex: 0,
        })
      }

      return res.json({
        syncing: highestSyncedL2Block < highestKnownL2Block,
        currentTransactionIndex: highestSyncedL2Block,
        highestKnownTransactionIndex: highestKnownL2Block,
      })
    })

    router.get('/eth/context/latest', async (req, res) => {
      const head = await this.options.l1RpcProvider.getBlockNumber()
      const safeHead = Math.max(0, head - this.options.confirmations)
      const block = await this.options.l1RpcProvider.getBlock(safeHead)
      if (block === null) {
        // Should not happen, since safeHead is always less than head and L1 RPC provider said the
        // head block exists. Could theoretically happen if using very low number of confirmations,
        // but no mainnet or testnet node would be running with few enough confirmations to make
        // this a potential problem.
        throw new Error(`cannot GET /eth/context/latest at ${safeHead}`)
      }

      return res.json({
        blockNumber: block.number,
        timestamp: block.timestamp,
        blockHash: block.hash,
      })
    })

    router.get('/eth/context/blocknumber/:number', async (req, res) => {
      const number = ethers.BigNumber.from(req.params.number).toNumber()
      const head = await this.options.l1RpcProvider.getBlockNumber()
      const safeHead = Math.max(0, head - this.options.confirmations)
      if (number > safeHead) {
        return res.json({
          blockNumber: null,
          timestamp: null,
          blockHash: null,
        })
      }

      const block = await this.options.l1RpcProvider.getBlock(number)
      if (block === null) {
        // Should not happen, same logic as above.
        throw new Error(`cannot GET /eth/context/blocknumber/${number}`)
      }

      return res.json({
        blockNumber: block.number,
        timestamp: block.timestamp,
        blockHash: block.hash,
      })
    })

    router.get('/enqueue/latest', async (req, res) => {
      const enqueue = await this.state.db.getEnqueue('latest')
      if (enqueue === null) {
        return res.json({
          index: null,
          target: null,
          data: null,
          gasLimit: null,
          origin: null,
          blockNumber: null,
          timestamp: null,
          ctcIndex: null,
        })
      }

      return res.json(enqueue)
    })

    router.get('/enqueue/index/:index', async (req, res) => {
      const enqueue = await this.state.db.getEnqueue(req.params.index)
      if (enqueue === null) {
        return res.json({
          index: null,
          target: null,
          data: null,
          gasLimit: null,
          origin: null,
          blockNumber: null,
          timestamp: null,
          ctcIndex: null,
        })
      }

      return res.json(enqueue)
    })

    router.get('/transaction/latest', async (req, res) => {
      const transaction = await this.state.db.getTransaction('latest')
      if (transaction === null) {
        return res.json({
          transaction: null,
          batch: null,
        })
      }

      const batch = await this.state.db.getBatch(transaction.batchIndex)
      if (batch === null) {
        return res.json({
          transaction: null,
          batch: null,
        })
      }

      return res.json({
        transaction,
        batch,
      })
    })

    router.get('/transaction/index/:index', async (req, res) => {
      const transaction = await this.state.db.getTransaction(req.params.index)
      if (transaction === null) {
        return res.json({
          transaction: null,
          batch: null,
        })
      }

      const batch = await this.state.db.getBatch(transaction.batchIndex)
      if (batch === null) {
        return res.json({
          transaction: null,
          batch: null,
        })
      }

      return res.json({
        transaction,
        batch,
      })
    })

    router.get('/batch/transaction/latest', async (req, res) => {
      const batch = await this.state.db.getBatch('latest')
      if (batch === null) {
        return res.json({
          batch: null,
          transactions: [],
        })
      }

      const transactions: BatchTransactionEntry[] = await Promise.all(
        range(batch.prevTotalElements, batch.size).map(async (index) => {
          return this.state.db.getTransaction(index)
        })
      )
      if (transactions.some((tx) => tx === null)) {
        return res.json({
          batch: null,
          transactions: [],
        })
      }

      return res.json({
        batch,
        transactions,
      })
    })

    router.get('/batch/transaction/index/:index', async (req, res) => {
      const batch = await this.state.db.getBatch(req.params.index)
      if (batch === null) {
        return res.json({
          batch: null,
          transactions: [],
        })
      }

      const transactions: BatchTransactionEntry[] = await Promise.all(
        range(batch.prevTotalElements, batch.size).map(async (index) => {
          return this.state.db.getTransaction(index)
        })
      )
      if (transactions.some((tx) => tx === null)) {
        return res.json({
          batch: null,
          transactions: [],
        })
      }

      return res.json({
        batch,
        transactions,
      })
    })
  }

  protected async init(): Promise<void> {
    // Set up DB connection.
    const db = level(this.options.db)
    await db.open()
    this.state.db = new TransportDB(new SimpleDB(db), this.options.l2ChainId)

    // Connect to the AddressManager, temporarily.
    const AddressManager = new ethers.Contract(
      this.options.addressManager,
      getContractInterface('Lib_AddressManager'),
      this.options.l1RpcProvider
    )

    // Use AddressManager to build CTC.
    this.state.CanonicalTransactionChain = new ethers.Contract(
      await this.retryable(
        AddressManager.getAddress.bind(
          AddressManager,
          'CanonicalTransactionChain'
        )
      ),
      getContractInterface('CanonicalTransactionChain'),
      this.options.l1RpcProvider
    )

    // Initialize highest synced L1 block if necessary.
    if (!(await this.state.db.getHighestSyncedL1Block())) {
      await this.state.db.putHighestSyncedL1Block(this.options.l1StartHeight)
    }

    // We cache the highest known L1 block to avoid making unnecessary requests. We're only going
    // to update this number if we actually sync all the way up to this block. This way we don't
    // need to query the latest block on every loop.
    this.state.highestKnownL1Block = await this.retryable(
      this.options.l1RpcProvider.getBlockNumber.bind(this.options.l1RpcProvider)
    )
  }

  protected async main(): Promise<void> {
    const highestSyncedL1Block = await this.state.db.getHighestSyncedL1Block()
    const syncRangeEnd = getRangeEnd(
      highestSyncedL1Block,
      this.state.highestKnownL1Block - this.options.confirmations,
      this.options.blocksPerLogQuery
    )

    if (highestSyncedL1Block === syncRangeEnd) {
      this.logger.info('synced to tip, checking for new L1 blocks')
      const latestL1Block = await this.retryable(
        this.options.l1RpcProvider.getBlockNumber.bind(
          this.options.l1RpcProvider
        )
      )
      if (latestL1Block > this.state.highestKnownL1Block) {
        this.logger.info('new L1 block found')
        this.state.highestKnownL1Block = latestL1Block
      } else {
        // Latest L1 block number hasn't updated yet and we've already synced all of the available
        // blocks so we'll just wait for the next iteration of the loop.
        this.logger.info('no new L1 blocks found, trying again in 15s')
        await sleep(15000)
        return
      }
    }

    // Update highest known L2 block. We do this *before* running the rest of the loop to avoid a
    // situation where there are elements in the DB that have an index higher than the recorded
    // highest known L2 block.
    const highestKnownL2Block = await this.retryable(
      this.state.CanonicalTransactionChain.getTotalElements.bind(
        this.state.CanonicalTransactionChain
      )
    )
    await this.state.db.putHighestKnownL2Block(highestKnownL2Block.toNumber())
    this.metrics.highestKnownL2Block.set(highestKnownL2Block.toNumber())

    try {
      await this.syncEventsFromCTC(
        highestSyncedL1Block,
        syncRangeEnd,
        'TransactionEnqueued',
        parseTransactionEnqueued
      )
    } catch (err) {
      if (err === ErrEntryInconsistency) {
        return
      } else {
        throw err
      }
    }

    try {
      await this.syncEventsFromCTC(
        highestSyncedL1Block,
        syncRangeEnd,
        'TransactionBatchAppended',
        parseTransactionBatchAppended
      )
    } catch (err) {
      if (err === ErrEntryInconsistency) {
        return
      } else {
        throw err
      }
    }

    // Now we record the highest synced L2 block. It's possible for the latest batch to be null
    // very early in the chain history if no batches have been found yet. Don't want to update the
    // highest synced L2 block if this is the case.
    const latestBatch = await this.state.db.getBatch('latest')
    if (latestBatch !== null) {
      const highestSyncedL2Block =
        latestBatch.prevTotalElements + latestBatch.size
      await this.state.db.putHighestSyncedL2Block(highestSyncedL2Block)
      this.metrics.highestSyncedL2Block.set(highestSyncedL2Block)
    }

    // If we made it all the way here then we successfully synced to the end of the range.
    await this.state.db.putHighestSyncedL1Block(syncRangeEnd)
    this.metrics.highestSyncedL1Block.set(syncRangeEnd)
  }

  public async syncEventsFromCTC(
    startBlock: number,
    endBlock: number,
    eventName: string,
    eventParsingFunction: EventParsingFunction
  ): Promise<void> {
    this.logger.info('started syncing events', {
      eventName,
      startBlock,
      endBlock,
    })

    const events = await this.retryable(
      this.state.CanonicalTransactionChain.queryFilter.bind(
        this.state.CanonicalTransactionChain,
        this.state.CanonicalTransactionChain.filters[eventName](),
        startBlock,
        endBlock
      )
    )

    this.logger.info('events to parse', {
      count: events.length,
    })

    for (let i = 0; i < events.length; i++) {
      const event = events[i]
      const entries = await this.retryable(
        eventParsingFunction.bind(
          this,
          event,
          this.options.l1RpcProvider,
          this.options.l2ChainId
        )
      )

      for (const entry of entries) {
        try {
          await this.state.db.db.put(entry.key, entry.index, entry)
        } catch (err) {
          if (err === ErrEntryInconsistency) {
            this.logger.warn('found event inconsistency, rolling back', {
              eventIndex: i,
              entry,
            })

            // Entry inconsistency happens when there's a missing entry between the latest entry in
            // the database and the entry we're trying to insert. This can happen when events are
            // not being properly returned by the remote L1 node, which is very rare but has
            // happened multiple times in the past. When we detect this, we reset the highest
            // synced L1 block to trigger a resync from the last good block.
            const latest = await this.state.db.db.get(entry.key, 'latest')
            const latestIndex = latest
              ? this.options.l1StartHeight
              : latest.index
            await this.state.db.putHighestSyncedL1Block(latestIndex)
          }
          throw err
        }
      }
    }

    this.logger.info(`finished syncing events`, {
      count: events.length,
    })
  }

  public retryable<TRet, T extends () => Promise<TRet>>(
    fn: T,
    opts: {
      name?: string
      max?: number
      backoff?: 'linear' | 'exponential'
    } = {}
  ): ReturnType<T> {
    return new Promise<TRet>(async (resolve, reject) => {
      const max = opts.max || 7
      let retries = max
      while (true) {
        try {
          const ret = await fn()
          resolve(ret)
          return
        } catch (err) {
          this.metrics.unexpectedRetryableErrors.inc({
            name: opts.name || fn.name,
          })

          const sleepTimeMs =
            opts.backoff === 'linear'
              ? 1000 * (max - retries)
              : 1000 * 2 ** (max - retries)
          this.logger.info(`caught unexpected error in retryable`, {
            retries,
            sleepTimeMs,
            error: err,
          })

          if (retries <= 0) {
            reject(err)
            return
          } else {
            await sleep(sleepTimeMs)
            retries--
          }
        }
      }
    }) as ReturnType<T>
  }
}

if (require.main === module) {
  const service = new DTLService()
  service.run()
}
