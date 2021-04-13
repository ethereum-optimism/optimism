/* Imports: External */
import { BaseService } from '@eth-optimism/core-utils'
import express, { Request, Response } from 'express'
import cors from 'cors'
import { BigNumber } from 'ethers'
import { JsonRpcProvider } from '@ethersproject/providers'
import { LevelUp } from 'levelup'

/* Imports: Internal */
import { TransportDB } from '../../db/transport-db'
import {
  ContextResponse,
  GasPriceResponse,
  EnqueueResponse,
  StateRootBatchResponse,
  StateRootResponse,
  SyncingResponse,
  TransactionBatchResponse,
  TransactionResponse,
} from '../../types'
import { validators } from '../../utils'
import { L1DataTransportServiceOptions } from '../main/service'

export interface L1TransportServerOptions
  extends L1DataTransportServiceOptions {
  db: LevelUp
}

const optionSettings = {
  db: {
    validate: validators.isLevelUP,
  },
  port: {
    default: 7878,
    validate: validators.isInteger,
  },
  hostname: {
    default: 'localhost',
    validate: validators.isString,
  },
  confirmations: {
    validate: validators.isInteger,
  },
  l1RpcProvider: {
    validate: (val: any) => {
      return validators.isUrl(val) || validators.isJsonRpcProvider(val)
    },
  },
  showUnconfirmedTransactions: {
    validate: validators.isBoolean,
  },
}

export class L1TransportServer extends BaseService<L1TransportServerOptions> {
  constructor(options: L1TransportServerOptions) {
    super('L1 Transport Server', options, optionSettings)
  }

  private state: {
    app: express.Express
    server: any
    db: TransportDB
    l1RpcProvider: JsonRpcProvider
  } = {} as any

  protected async _init(): Promise<void> {
    // TODO: I don't know if this is strictly necessary, but it's probably a good thing to do.
    if (!this.options.db.isOpen()) {
      await this.options.db.open()
    }

    this.state.db = new TransportDB(this.options.db)
    this.state.l1RpcProvider =
      typeof this.options.l1RpcProvider === 'string'
        ? new JsonRpcProvider(this.options.l1RpcProvider)
        : this.options.l1RpcProvider

    this._initializeApp()
  }

  protected async _start(): Promise<void> {
    this.state.server = this.state.app.listen(
      this.options.port,
      this.options.hostname
    )
    this.logger.info('Server started and listening', {
      host: this.options.hostname,
      port: this.options.port,
    })
  }

  protected async _stop(): Promise<void> {
    this.state.server.close()
  }

  /**
   * Initializes the server application.
   * Do any sort of initialization here that you want. Mostly just important that
   * `_registerAllRoutes` is called at the end.
   */
  private _initializeApp() {
    // TODO: Maybe pass this in as a parameter instead of creating it here?
    this.state.app = express()
    this.state.app.use(cors())
    this._registerAllRoutes()
    this.logger.info('All routes registered for L1 Transport Server')
  }

  /**
   * Registers a route on the server.
   * @param method Http method type.
   * @param route Route to register.
   * @param handler Handler called and is expected to return a JSON response.
   */
  private _registerRoute(
    method: 'get', // Just handle GET for now, but could extend this with whatever.
    route: string,
    handler: (req?: Request, res?: Response) => Promise<any>
  ): void {
    // TODO: Better typing on the return value of the handler function.
    // TODO: Check for route collisions.
    // TODO: Add a different function to allow for removing routes.

    this.state.app[method](route, async (req, res) => {
      const start = Date.now()
      try {
        const json = await handler(req, res)
        const elapsed = Date.now() - start
        this.logger.info('Served HTTP Request', {
          method: req.method,
          url: req.url,
          elapsed,
        })
        return res.json(json)
      } catch (e) {
        const elapsed = Date.now() - start
        this.logger.info('Failed HTTP Request', {
          method: req.method,
          url: req.url,
          elapsed,
          msg: e.toString(),
        })
        return res.status(400).json({
          error: e.toString(),
        })
      }
    })
  }

  /**
   * Registers all of the server routes we want to expose.
   * TODO: Link to our API spec.
   */
  private _registerAllRoutes(): void {
    // TODO: Maybe add doc-like comments to each of these routes?

    this._registerRoute(
      'get',
      '/eth/syncing',
      async (): Promise<SyncingResponse> => {
        this.logger.info('Retrieving L2 syncing status...')
        const highestL2BlockNumber = await this.state.db.getHighestL2BlockNumber()
        this.logger.info('Got highest L2 block number', {
          highestL2BlockNumber,
        })
        const currentL2Block = await this.state.db.getLatestTransaction()
        this.logger.info('Got current L2 block', {
          currentL2Block,
        })

        if (currentL2Block === null) {
          if (highestL2BlockNumber === null) {
            return {
              syncing: false,
              currentTransactionIndex: 0,
            }
          } else {
            return {
              syncing: true,
              highestKnownTransactionIndex: highestL2BlockNumber,
              currentTransactionIndex: 0,
            }
          }
        }

        if (highestL2BlockNumber > currentL2Block.index) {
          return {
            syncing: true,
            highestKnownTransactionIndex: highestL2BlockNumber,
            currentTransactionIndex: currentL2Block.index,
          }
        } else {
          return {
            syncing: false,
            currentTransactionIndex: currentL2Block.index,
          }
        }
      }
    )

    this._registerRoute(
      'get',
      '/eth/gasprice',
      async (): Promise<GasPriceResponse> => {
        this.logger.info('Retrieving L1 gas price...')
        const gasPrice = await this.state.l1RpcProvider.getGasPrice()
        this.logger.info('Got L1 gas price', {
          gasPrice,
        })

        return {
          gasPrice: gasPrice.toString(),
        }
      }
    )

    this._registerRoute(
      'get',
      '/eth/context/latest',
      async (): Promise<ContextResponse> => {
        this.logger.info('Retrieving latest L1 context...')
        const tip = await this.state.l1RpcProvider.getBlockNumber()
        this.logger.info('Got L1 tip block number', {
          tip,
        })
        const blockNumber = Math.max(0, tip - this.options.confirmations)

        const block = await this.state.l1RpcProvider.getBlock(blockNumber)
        this.logger.info('Got L1 tip block', {
          blockNumber: block.number,
          timestamp: block.timestamp,
          blockHash: block.hash,
        })

        return {
          blockNumber: block.number,
          timestamp: block.timestamp,
          blockHash: block.hash,
        }
      }
    )

    this._registerRoute(
      'get',
      '/eth/context/blocknumber/:number',
      async (req): Promise<ContextResponse> => {
        const number = BigNumber.from(req.params.number).toNumber()
        this.logger.info('Retrieving L1 block by number...', {
          number,
        })
        const tip = await this.state.l1RpcProvider.getBlockNumber()
        this.logger.info('Got L1 chain tip block number', {
          tip,
        })
        const blockNumber = Math.max(0, tip - this.options.confirmations)

        if (number > blockNumber) {
          this.logger.info(
            'Requested block number is not confirmed, returning null block'
          )
          return {
            blockNumber: null,
            timestamp: null,
            blockHash: null,
          }
        }

        const block = await this.state.l1RpcProvider.getBlock(number)
        this.logger.info('Got L1 block by number', {
          blockNumber: block.number,
          timestamp: block.timestamp,
          blockHash: block.hash,
        })
        return {
          blockNumber: block.number,
          timestamp: block.timestamp,
          blockHash: block.hash,
        }
      }
    )

    this._registerRoute(
      'get',
      '/enqueue/latest',
      async (): Promise<EnqueueResponse> => {
        this.logger.info('Retrieving latest enqueue...')
        const enqueue = await this.state.db.getLatestEnqueue()
        this.logger.info('Got latest enqueue', {
          enqueue,
        })

        if (enqueue === null) {
          return {
            index: null,
            target: null,
            data: null,
            gasLimit: null,
            origin: null,
            blockNumber: null,
            timestamp: null,
            ctcIndex: null,
          }
        }

        const ctcIndex = await this.state.db.getTransactionIndexByQueueIndex(
          enqueue.index
        )
        this.logger.info('Got transaction index at queue index', {
          queueIndex: enqueue.index,
          ctcIndex,
        })

        return {
          ...enqueue,
          ctcIndex,
        }
      }
    )

    this._registerRoute(
      'get',
      '/enqueue/index/:index',
      async (req): Promise<EnqueueResponse> => {
        const index = BigNumber.from(req.params.index).toNumber()
        this.logger.info('Retrieving equeue by index...', {
          index,
        })
        const enqueue = await this.state.db.getEnqueueByIndex(index)
        this.logger.info('Got enqueue at index', {
          index,
          enqueue,
        })

        if (enqueue === null) {
          return {
            index: null,
            target: null,
            data: null,
            gasLimit: null,
            origin: null,
            blockNumber: null,
            timestamp: null,
            ctcIndex: null,
          }
        }

        const ctcIndex = await this.state.db.getTransactionIndexByQueueIndex(
          enqueue.index
        )
        this.logger.info('Got transaction index at queue index', {
          queueIndex: enqueue.index,
          ctcIndex,
        })

        return {
          ...enqueue,
          ctcIndex,
        }
      }
    )

    this._registerRoute(
      'get',
      '/transaction/latest',
      async (): Promise<TransactionResponse> => {
        this.logger.info('Retrieving latest transaction...')
        let transaction = await this.state.db.getLatestFullTransaction()
        this.logger.info('Got latest transaction', {
          transaction,
        })

        if (this.options.showUnconfirmedTransactions) {
          const latestUnconfirmedTx = await this.state.db.getLatestUnconfirmedTransaction()
          this.logger.info('Got latest unconfirmed transaction', {
            latestUnconfirmedTx,
          })
          if (
            transaction === null ||
            transaction === undefined ||
            latestUnconfirmedTx.index >= transaction.index
          ) {
            transaction = latestUnconfirmedTx
          }
        }

        if (transaction === null) {
          this.logger.info('Latest transaction was null, retrying...')
          transaction = await this.state.db.getLatestFullTransaction()
          this.logger.info('Retried and got latest transaction', {
            transaction,
          })
        }

        if (transaction === null) {
          return {
            transaction: null,
            batch: null,
          }
        }

        const batch = await this.state.db.getTransactionBatchByIndex(
          transaction.batchIndex
        )
        this.logger.info('Got transaction batch', {
          batchIndex: transaction.batchIndex,
          batch,
        })

        return {
          transaction,
          batch,
        }
      }
    )

    this._registerRoute(
      'get',
      '/transaction/index/:index',
      async (req): Promise<TransactionResponse> => {
        let transaction = null
        const index = BigNumber.from(req.params.index).toNumber()
        this.logger.info('Retrieving transaction by index...', {
          index,
        })
        if (this.options.showUnconfirmedTransactions) {
          transaction = await this.state.db.getUnconfirmedTransactionByIndex(
            index
          )
          this.logger.info('Got latest unconfirmed transaction', {
            transaction,
          })
        }

        if (transaction === null) {
          transaction = await this.state.db.getFullTransactionByIndex(index)
          this.logger.info('Got latest full transaction', {
            transaction,
          })
        }

        if (transaction === null) {
          return {
            transaction: null,
            batch: null,
          }
        }

        const batch = await this.state.db.getTransactionBatchByIndex(
          transaction.batchIndex
        )
        this.logger.info('Got transaction batch', {
          batchIndex: transaction.batchIndex,
          batch,
        })

        return {
          transaction,
          batch,
        }
      }
    )

    this._registerRoute(
      'get',
      '/batch/transaction/latest',
      async (): Promise<TransactionBatchResponse> => {
        this.logger.info('Retrieving latest transaction batch...')
        const batch = await this.state.db.getLatestTransactionBatch()
        this.logger.info('Got latest transaction batch', {
          batch,
        })

        if (batch === null) {
          return {
            batch: null,
            transactions: [],
          }
        }

        const start = BigNumber.from(batch.prevTotalElements).toNumber()
        const end =
          BigNumber.from(batch.prevTotalElements).toNumber() +
          BigNumber.from(batch.size).toNumber()

        const transactions = await this.state.db.getFullTransactionsByIndexRange(
          start,
          end
        )
        this.logger.info('Got transactions in batch', {
          start,
          end,
        })

        return {
          batch,
          transactions,
        }
      }
    )

    this._registerRoute(
      'get',
      '/batch/transaction/index/:index',
      async (req): Promise<TransactionBatchResponse> => {
        const index = BigNumber.from(req.params.index).toNumber()
        this.logger.info('Retrieving transaction batch by index...', {
          index,
        })
        const batch = await this.state.db.getTransactionBatchByIndex(index)
        this.logger.info('Got transaction batch by index', {
          index,
          batch,
        })

        if (batch === null) {
          return {
            batch: null,
            transactions: [],
          }
        }

        const start = BigNumber.from(batch.prevTotalElements).toNumber()
        const end =
          BigNumber.from(batch.prevTotalElements).toNumber() +
          BigNumber.from(batch.size).toNumber()
        const transactions = await this.state.db.getFullTransactionsByIndexRange(
          start,
          end
        )
        this.logger.info('Got transactions in batch', {
          start,
          end,
        })

        return {
          batch,
          transactions,
        }
      }
    )

    this._registerRoute(
      'get',
      '/stateroot/latest',
      async (): Promise<StateRootResponse> => {
        this.logger.info('Retrieving latest state root...')
        let stateRoot = await this.state.db.getLatestStateRoot()
        this.logger.info('Got latest state root', {
          stateRoot,
        })
        if (this.options.showUnconfirmedTransactions) {
          const latestUnconfirmedStateRoot = await this.state.db.getLatestUnconfirmedStateRoot()
          this.logger.info('Got latest unconfirmed state root', {
            latestUnconfirmedStateRoot,
          })
          if (
            stateRoot === null ||
            stateRoot === undefined ||
            latestUnconfirmedStateRoot.index >= stateRoot.index
          ) {
            stateRoot = latestUnconfirmedStateRoot
          }
        }

        if (stateRoot === null) {
          this.logger.info('Latest transaction was null, retrying...')
          stateRoot = await this.state.db.getLatestStateRoot()
          this.logger.info('Retried and got latest state root', {
            stateRoot,
          })
        }

        if (stateRoot === null) {
          return {
            stateRoot: null,
            batch: null,
          }
        }

        const batch = await this.state.db.getStateRootBatchByIndex(
          stateRoot.batchIndex
        )
        this.logger.info('Got state root batch', {
          batchIndex: stateRoot.batchIndex,
          batch,
        })

        return {
          stateRoot,
          batch,
        }
      }
    )

    this._registerRoute(
      'get',
      '/stateroot/index/:index',
      async (req): Promise<StateRootResponse> => {
        let stateRoot = null
        const index = BigNumber.from(req.params.index).toNumber()
        this.logger.info('Retrieving state root by index...', {
          index,
        })
        if (this.options.showUnconfirmedTransactions) {
          stateRoot = await this.state.db.getUnconfirmedStateRootByIndex(index)
          this.logger.info('Got unconfirmed state root by index', {
            index,
            stateRoot,
          })
        }

        if (stateRoot === null) {
          stateRoot = await this.state.db.getStateRootByIndex(index)
          this.logger.info('Got state root by index', {
            index,
            stateRoot,
          })
        }

        if (stateRoot === null) {
          return {
            stateRoot: null,
            batch: null,
          }
        }

        const batch = await this.state.db.getStateRootBatchByIndex(
          stateRoot.batchIndex
        )
        this.logger.info('Got state root batch', {
          batchIndex: stateRoot.batchIndex,
          batch,
        })

        return {
          stateRoot,
          batch,
        }
      }
    )

    this._registerRoute(
      'get',
      '/batch/stateroot/latest',
      async (): Promise<StateRootBatchResponse> => {
        this.logger.info('Retrieving latest state root batch...')
        const batch = await this.state.db.getLatestStateRootBatch()
        this.logger.info('Got latest state root batch', {
          batch,
        })

        if (batch === null) {
          return {
            batch: null,
            stateRoots: [],
          }
        }

        const start = BigNumber.from(batch.prevTotalElements).toNumber()
        const end =
          BigNumber.from(batch.prevTotalElements).toNumber() +
          BigNumber.from(batch.size).toNumber()
        const stateRoots = await this.state.db.getStateRootsByIndexRange(
          start,
          end
        )
        this.logger.info('Got state roots in batch', {
          start,
          end,
        })

        return {
          batch,
          stateRoots,
        }
      }
    )

    this._registerRoute(
      'get',
      '/batch/stateroot/index/:index',
      async (req): Promise<StateRootBatchResponse> => {
        const index = BigNumber.from(req.params.index).toNumber()
        this.logger.info('Retrieving state root batch by index...', {
          index,
        })
        const batch = await this.state.db.getStateRootBatchByIndex(index)
        this.logger.info('Got state root batch by index', {
          index,
          batch,
        })

        if (batch === null) {
          return {
            batch: null,
            stateRoots: [],
          }
        }

        const start = BigNumber.from(batch.prevTotalElements).toNumber()
        const end =
          BigNumber.from(batch.prevTotalElements).toNumber() +
          BigNumber.from(batch.size).toNumber()
        const stateRoots = await this.state.db.getStateRootsByIndexRange(
          start,
          end
        )
        this.logger.info('Got state roots in batch', {
          start,
          end,
        })

        return {
          batch,
          stateRoots,
        }
      }
    )
  }
}
