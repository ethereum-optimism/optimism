/* Imports: External */
import { BaseService, Logger, LegacyMetrics } from '@eth-optimism/common-ts'
import express, { Request, Response } from 'express'
import promBundle from 'express-prom-bundle'
import cors from 'cors'
import { BigNumber } from 'ethers'
import { JsonRpcProvider } from '@ethersproject/providers'
import { LevelUp } from 'levelup'
import * as Sentry from '@sentry/node'
import * as Tracing from '@sentry/tracing'

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
  metrics: LegacyMetrics
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
  l2RpcProvider: {
    validate: (val: unknown) => {
      return validators.isUrl(val) || validators.isJsonRpcProvider(val)
    },
  },
  defaultBackend: {
    default: 'l1',
    validate: (val: string) => {
      return val === 'l1' || val === 'l2'
    },
  },
  l1GasPriceBackend: {
    default: 'l1',
    validate: (val: string) => {
      return val === 'l1' || val === 'l2'
    },
  },
}

export class L1TransportServer extends BaseService<L1TransportServerOptions> {
  constructor(options: L1TransportServerOptions) {
    super('L1_Transport_Server', options, optionSettings)
  }

  private state: {
    app: express.Express
    server: any
    db: TransportDB
    l1RpcProvider: JsonRpcProvider
    l2RpcProvider: JsonRpcProvider
  } = {} as any

  protected async _init(): Promise<void> {
    if (!this.options.db.isOpen()) {
      await this.options.db.open()
    }

    this.state.db = new TransportDB(this.options.db, {
      l2ChainId: this.options.l2ChainId,
    })

    this.state.l1RpcProvider =
      typeof this.options.l1RpcProvider === 'string'
        ? new JsonRpcProvider({
            url: this.options.l1RpcProvider,
            user: this.options.l1RpcProviderUser,
            password: this.options.l1RpcProviderPassword,
            headers: { 'User-Agent': 'data-transport-layer' },
          })
        : this.options.l1RpcProvider

    this.state.l2RpcProvider =
      typeof this.options.l2RpcProvider === 'string'
        ? new JsonRpcProvider({
            url: this.options.l2RpcProvider,
            user: this.options.l2RpcProviderUser,
            password: this.options.l2RpcProviderPassword,
            headers: { 'User-Agent': 'data-transport-layer' },
          })
        : this.options.l2RpcProvider

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

    if (this.options.useSentry) {
      this._initSentry()
    }

    this.state.app.use(cors())

    // Add prometheus middleware to express BEFORE route registering
    this.state.app.use(
      // This also serves metrics on port 3000 at /metrics
      promBundle({
        // Provide metrics registry that other metrics uses
        promRegistry: this.metrics.registry,
        includeMethod: true,
        includePath: true,
      })
    )

    this._registerAllRoutes()

    // Sentry error handling must be after all controllers
    // and before other error middleware
    if (this.options.useSentry) {
      this.state.app.use(Sentry.Handlers.errorHandler())
    }

    this.logger.info('HTTP Server Options', {
      defaultBackend: this.options.defaultBackend,
      l1GasPriceBackend: this.options.l1GasPriceBackend,
    })

    if (this.state.l1RpcProvider) {
      this.logger.info('HTTP Server L1 RPC Provider initialized', {
        url: this.state.l1RpcProvider.connection.url,
      })
    } else {
      this.logger.warn('HTTP Server L1 RPC Provider not initialized')
    }
    if (this.state.l2RpcProvider) {
      this.logger.info('HTTP Server L2 RPC Provider initialized', {
        url: this.state.l2RpcProvider.connection.url,
      })
    } else {
      this.logger.warn('HTTP Server L2 RPC Provider not initialized')
    }
  }

  /**
   * Initialize Sentry and related middleware
   */
  private _initSentry() {
    const sentryOptions = {
      dsn: this.options.sentryDsn,
      release: this.options.release,
      environment: this.options.ethNetworkName,
    }
    this.logger = new Logger({
      name: this.name,
      sentryOptions,
    })
    Sentry.init({
      ...sentryOptions,
      integrations: [
        new Sentry.Integrations.Http({ tracing: true }),
        new Tracing.Integrations.Express({
          app: this.state.app,
        }),
      ],
      tracesSampleRate: this.options.sentryTraceRate,
    })
    this.state.app.use(Sentry.Handlers.requestHandler())
    this.state.app.use(Sentry.Handlers.tracingHandler())
  }

  /**
   * Registers a route on the server.
   *
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
        this.logger.debug('Response body', {
          method: req.method,
          url: req.url,
          body: json,
        })
        return res.json(json)
      } catch (e) {
        const elapsed = Date.now() - start
        this.logger.error('Failed HTTP Request', {
          method: req.method,
          url: req.url,
          elapsed,
          msg: e.toString(),
          stack: e.stack,
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
    this._registerRoute(
      'get',
      '/eth/syncing',
      async (req): Promise<SyncingResponse> => {
        const backend = req.query.backend || this.options.defaultBackend

        let currentL2Block
        let highestL2BlockNumber
        switch (backend) {
          case 'l1':
            currentL2Block = await this.state.db.getLatestTransaction()
            highestL2BlockNumber = await this.state.db.getHighestL2BlockNumber()
            break
          case 'l2':
            currentL2Block =
              await this.state.db.getLatestUnconfirmedTransaction()
            highestL2BlockNumber =
              (await this.state.db.getHighestSyncedUnconfirmedBlock()) - 1
            break
          default:
            throw new Error(`Unknown transaction backend ${backend}`)
        }

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
      async (req): Promise<GasPriceResponse> => {
        const backend = req.query.backend || this.options.l1GasPriceBackend
        let gasPrice: BigNumber

        if (backend === 'l1') {
          gasPrice = await this.state.l1RpcProvider.getGasPrice()
        } else if (backend === 'l2') {
          const response = await this.state.l2RpcProvider.send(
            'rollup_gasPrices',
            []
          )
          gasPrice = BigNumber.from(response.l1GasPrice)
        } else {
          throw new Error(`Unknown L1 gas price backend: ${backend}`)
        }

        return {
          gasPrice: gasPrice.toString(),
        }
      }
    )

    this._registerRoute(
      'get',
      '/eth/context/latest',
      async (): Promise<ContextResponse> => {
        const tip = await this.state.l1RpcProvider.getBlockNumber()
        const blockNumber = Math.max(0, tip - this.options.confirmations)

        const block = await this.state.l1RpcProvider.getBlock(blockNumber)
        if (block === null) {
          throw new Error(`Cannot GET /eth/context/latest at ${blockNumber}`)
        }

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
        const tip = await this.state.l1RpcProvider.getBlockNumber()
        const blockNumber = Math.max(0, tip - this.options.confirmations)

        if (number > blockNumber) {
          return {
            blockNumber: null,
            timestamp: null,
            blockHash: null,
          }
        }

        const block = await this.state.l1RpcProvider.getBlock(number)
        if (block === null) {
          throw new Error(`Cannot GET /eth/context/blocknumber/${number}`)
        }

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
        const enqueue = await this.state.db.getLatestEnqueue()

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
        const enqueue = await this.state.db.getEnqueueByIndex(
          BigNumber.from(req.params.index).toNumber()
        )

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

        return {
          ...enqueue,
          ctcIndex,
        }
      }
    )

    this._registerRoute(
      'get',
      '/transaction/latest',
      async (req): Promise<TransactionResponse> => {
        const backend = req.query.backend || this.options.defaultBackend
        let transaction = null

        switch (backend) {
          case 'l1':
            transaction = await this.state.db.getLatestFullTransaction()
            break
          case 'l2':
            transaction = await this.state.db.getLatestUnconfirmedTransaction()
            break
          default:
            throw new Error(`Unknown transaction backend ${backend}`)
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
        const backend = req.query.backend || this.options.defaultBackend
        let transaction = null

        switch (backend) {
          case 'l1':
            transaction = await this.state.db.getFullTransactionByIndex(
              BigNumber.from(req.params.index).toNumber()
            )
            break
          case 'l2':
            transaction = await this.state.db.getUnconfirmedTransactionByIndex(
              BigNumber.from(req.params.index).toNumber()
            )
            break
          default:
            throw new Error(`Unknown transaction backend ${backend}`)
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
        const batch = await this.state.db.getLatestTransactionBatch()

        if (batch === null) {
          return {
            batch: null,
            transactions: [],
          }
        }

        const transactions =
          await this.state.db.getFullTransactionsByIndexRange(
            BigNumber.from(batch.prevTotalElements).toNumber(),
            BigNumber.from(batch.prevTotalElements).toNumber() +
              BigNumber.from(batch.size).toNumber()
          )

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
        const batch = await this.state.db.getTransactionBatchByIndex(
          BigNumber.from(req.params.index).toNumber()
        )

        if (batch === null) {
          return {
            batch: null,
            transactions: [],
          }
        }

        const transactions =
          await this.state.db.getFullTransactionsByIndexRange(
            BigNumber.from(batch.prevTotalElements).toNumber(),
            BigNumber.from(batch.prevTotalElements).toNumber() +
              BigNumber.from(batch.size).toNumber()
          )

        return {
          batch,
          transactions,
        }
      }
    )

    this._registerRoute(
      'get',
      '/stateroot/latest',
      async (req): Promise<StateRootResponse> => {
        const backend = req.query.backend || this.options.defaultBackend
        let stateRoot = null

        switch (backend) {
          case 'l1':
            stateRoot = await this.state.db.getLatestStateRoot()
            break
          case 'l2':
            stateRoot = await this.state.db.getLatestUnconfirmedStateRoot()
            break
          default:
            throw new Error(`Unknown transaction backend ${backend}`)
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
        const backend = req.query.backend || this.options.defaultBackend
        let stateRoot = null

        switch (backend) {
          case 'l1':
            stateRoot = await this.state.db.getStateRootByIndex(
              BigNumber.from(req.params.index).toNumber()
            )
            break
          case 'l2':
            stateRoot = await this.state.db.getUnconfirmedStateRootByIndex(
              BigNumber.from(req.params.index).toNumber()
            )
            break
          default:
            throw new Error(`Unknown transaction backend ${backend}`)
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
        const batch = await this.state.db.getLatestStateRootBatch()

        if (batch === null) {
          return {
            batch: null,
            stateRoots: [],
          }
        }

        const stateRoots = await this.state.db.getStateRootsByIndexRange(
          BigNumber.from(batch.prevTotalElements).toNumber(),
          BigNumber.from(batch.prevTotalElements).toNumber() +
            BigNumber.from(batch.size).toNumber()
        )

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
        const batch = await this.state.db.getStateRootBatchByIndex(
          BigNumber.from(req.params.index).toNumber()
        )

        if (batch === null) {
          return {
            batch: null,
            stateRoots: [],
          }
        }

        const stateRoots = await this.state.db.getStateRootsByIndexRange(
          BigNumber.from(batch.prevTotalElements).toNumber(),
          BigNumber.from(batch.prevTotalElements).toNumber() +
            BigNumber.from(batch.size).toNumber()
        )

        return {
          batch,
          stateRoots,
        }
      }
    )
  }
}
