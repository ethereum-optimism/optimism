/* Imports: External */
import { ethers } from 'ethers'
import {
  BaseServiceV2,
  ExpressRouter,
  validators,
} from '@eth-optimism/common-ts'
import { Provider } from '@ethersproject/abstract-provider'

type DTLOptions = {
  l1RpcProvider: Provider
  l2RpcProvider: Provider
  l1StartHeight: number
  canSyncUnconfirmedTransactions: boolean
  numConfirmations: number
}

type DTLMetrics = {}

type DTLState = {
  // TODO: Fix this
  db: any
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
        l2RpcProvider: {
          validator: validators.provider,
          desc: 'provider for interacting with L2',
          default: new ethers.providers.JsonRpcProvider(),
          secret: true,
        },
        l1StartHeight: {
          validator: validators.num,
          desc: 'L1 block height where the L2 chain starts',
        },
        canSyncUnconfirmedTransactions: {
          validator: validators.bool,
          desc: 'whether or not to sync unconfirmed blocks from L2',
          default: true,
        },
        numConfirmations: {
          validator: validators.num,
          desc: 'number of confirmations when syncing from L1',
        },
      },
      metricsSpec: {},
    })
  }

  protected async init(): Promise<void> {}

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
      const safeHead = Math.max(0, head - this.options.numConfirmations)

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
      const safeHead = Math.max(0, head - this.options.numConfirmations)

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

  protected async main(): Promise<void> {}
}

if (require.main === module) {
  const service = new DTLService()
  service.run()
}
