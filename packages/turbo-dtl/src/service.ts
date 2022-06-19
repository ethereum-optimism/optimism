/* Imports: External */
import { ethers } from 'ethers'
import {
  BaseServiceV2,
  ExpressRouter,
  validators,
} from '@eth-optimism/common-ts'
import { Provider } from '@ethersproject/abstract-provider'
import { getContractInterface } from '@eth-optimism/contracts'

import {
  parseTransactionEnqueued,
  parseTransactionBatchAppended,
  EventParsingFunction,
} from './events'
import { ErrEntryInconsistency } from './consistency'
import { Keys } from './db'

type DTLOptions = {
  l1RpcProvider: Provider
  l2ChainId: number
  l1StartHeight: number
  confirmations: number
  blocksPerLogQuery: number
  addressManager: string
}

type DTLMetrics = {}

type DTLState = {
  // TODO: Fix this
  db: any
  highestKnownL1Block: number
  AddressManager: ethers.Contract
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
      name: 'dtl',
      options,
      optionsSpec: {
        l1RpcProvider: {
          validator: validators.provider,
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
      metricsSpec: {},
    })
  }

  protected async routes(router: ExpressRouter): Promise<void> {
    router.get('/eth/syncing', async (req, res) => {
      const highestSyncedL2Block = await this.state.db.get(
        keys.HIGHEST_SYNCED_L2_BLOCK_KEY
      )
      const highestKnownL2Block = await this.state.db.get(
        keys.HIGHEST_KNOWN_L2_BLOCK_KEY
      )

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
      const enqueue = await this.state.db.get(
        keys.ENQUEUE_TRANSACTION_KEY,
        'latest'
      )

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
      } else {
        return res.json(enqueue)
      }
    })

    router.get('/enqueue/index/:index', async (req, res) => {
      const enqueue = await this.state.db.get(
        keys.ENQUEUE_TRANSACTION_KEY,
        req.params.index
      )

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
      } else {
        return res.json(enqueue)
      }
    })

    router.get('/transaction/latest', async (req, res) => {
      const transaction = await this.state.db.get(
        keys.CHAIN_TRANSACTION_KEY,
        'latest'
      )

      if (transaction === null) {
        return res.json({
          transaction: null,
          batch: null,
        })
      }

      const batch = await this.state.db.get(
        keys.TRANSACTION_BATCH_KEY,
        transaction.batchIndex
      )

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
      const transaction = await this.state.db.get(
        keys.CHAIN_TRANSACTION_KEY,
        req.params.index
      )

      if (transaction === null) {
        return res.json({
          transaction: null,
          batch: null,
        })
      }

      const batch = await this.state.db.get(
        keys.TRANSACTION_BATCH_KEY,
        transaction.batchIndex
      )

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
      const batch = await this.state.db.get(
        keys.TRANSACTION_BATCH_KEY,
        'latest'
      )

      if (batch === null) {
        return res.json({
          batch: null,
          transactions: [],
        })
      }

      const transactions = await this.state.db.get(
        keys.CHAIN_TRANSACTION_KEY,
        batch.prevTotalElements,
        batch.prevTotalElements + batch.size
      )

      return res.json({
        batch,
        transactions,
      })
    })

    router.get('/batch/transaction/index/:index', async (req, res) => {
      const batch = await this.state.db.get(
        keys.TRANSACTION_BATCH_KEY,
        req.params.index
      )

      if (batch === null) {
        return res.json({
          batch: null,
          transactions: [],
        })
      }

      const transactions = await this.state.db.get(
        keys.CHAIN_TRANSACTION_KEY,
        batch.prevTotalElements,
        batch.prevTotalElements + batch.size
      )

      return res.json({
        batch,
        transactions,
      })
    })
  }

  protected async init(): Promise<void> {
    // Connect to the AddressManager and CTC.
    this.state.AddressManager = new ethers.Contract(
      this.options.addressManager,
      getContractInterface('Lib_AddressManager'),
      this.options.l1RpcProvider
    )
    this.state.CanonicalTransactionChain = new ethers.Contract(
      await this.state.AddressManager.getAddress('CanonicalTransactionChain'),
      getContractInterface('CanonicalTransactionChain'),
      this.options.l1RpcProvider
    )

    // Initialize the highest synced L1 block number if necessary.
    const highestSyncedL1Block = await this.state.db.get(
      keys.HIGHEST_SYNCED_L1_BLOCK_KEY
    )
    if (highestSyncedL1Block === null) {
      await this.state.db.put(
        keys.HIGHEST_SYNCED_L1_BLOCK_KEY,
        this.options.l1StartHeight
      )
    }

    // We cache the highest known L1 block to avoid making unnecessary requests. We're only going
    // to update this number if we actually sync all the way up to this block. This way we don't
    // need to query the latest block on every loop.
    this.state.highestKnownL1Block =
      await this.options.l1RpcProvider.getBlockNumber()
  }

  protected async main(): Promise<void> {
    const highestSyncedL1Block = await this.state.db.get(
      Keys.HIGHEST_SYNCED_L1_BLOCK
    )

    // Don't try to sync past the allowable tip based on the number of confirmations.
    const syncRangeEndBlock = Math.min(
      highestSyncedL1Block + this.options.blocksPerLogQuery,
      Math.max(0, this.state.highestKnownL1Block - this.options.confirmations)
    )

    if (highestSyncedL1Block === syncRangeEndBlock) {
      const latestL1Block = await this.options.l1RpcProvider.getBlockNumber()
      if (latestL1Block > this.state.highestKnownL1Block) {
        this.state.highestKnownL1Block = latestL1Block
      } else {
        // Latest L1 block number hasn't updated yet and we've already synced all of the available
        // blocks so we'll just wait for the next iteration of the loop.
        // TODO: Sleep here.
        return
      }
    }

    try {
      await this.syncEventsFromCTC(
        highestSyncedL1Block,
        syncRangeEndBlock,
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
        syncRangeEndBlock,
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
  }

  public async syncEventsFromCTC(
    startBlock: number,
    endBlock: number,
    eventName: string,
    eventParsingFunction: EventParsingFunction
  ): Promise<void> {
    const entries = await eventParsingFunction(
      await this.state.CanonicalTransactionChain.queryFilter(
        this.state.CanonicalTransactionChain.filters[eventName](),
        startBlock,
        endBlock
      ),
      this.options.l1RpcProvider,
      this.options.l2ChainId
    )

    for (const entry of entries) {
      try {
        await this.state.db.put(entry.key, entry.index, entry)
      } catch (err) {
        if (err === ErrEntryInconsistency) {
          const latest = await this.state.db.get(entry.key, 'latest')
          if (latest === null) {
            await this.state.db.put(
              Keys.HIGHEST_SYNCED_L1_BLOCK,
              this.options.l1StartHeight
            )
          } else {
            await this.state.db.put(Keys.HIGHEST_SYNCED_L1_BLOCK, latest.index)
          }
        }

        throw err
      }
    }
  }
}

if (require.main === module) {
  const service = new DTLService()
  service.run()
}
