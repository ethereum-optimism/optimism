/* Imports: External */
import { BaseService, LegacyMetrics } from '@eth-optimism/common-ts'
import { LevelUp } from 'levelup'
import level from 'level6'
import { Counter } from 'prom-client'

/* Imports: Internal */
import { L1IngestionService } from '../l1-ingestion/service'
import { L1TransportServer } from '../server/service'
import { validators } from '../../utils'
import { L2IngestionService } from '../l2-ingestion/service'
import { BSS_HF1_INDEX } from '../../config'

export interface L1DataTransportServiceOptions {
  nodeEnv: string
  ethNetworkName?: 'mainnet' | 'kovan' | 'goerli'
  release: string
  addressManager: string
  confirmations: number
  dangerouslyCatchAllErrors?: boolean
  hostname: string
  l1RpcProvider: string
  l1RpcProviderUser?: string
  l1RpcProviderPassword?: string
  l2ChainId: number
  l2RpcProvider: string
  l2RpcProviderUser?: string
  l2RpcProviderPassword?: string
  l1SyncShutoffBlock?: number
  metrics?: LegacyMetrics
  dbPath: string
  logsPerPollingInterval: number
  pollingInterval: number
  port: number
  syncFromL1?: boolean
  syncFromL2?: boolean
  transactionsPerPollingInterval: number
  legacySequencerCompatibility: boolean
  useSentry?: boolean
  sentryDsn?: string
  sentryTraceRate?: number
  defaultBackend: string
  l1GasPriceBackend: string
  l1StartHeight?: number
}

const optionSettings = {
  syncFromL1: {
    default: true,
    validate: validators.isBoolean,
  },
  syncFromL2: {
    default: false,
    validate: validators.isBoolean,
  },
}

// prettier-ignore
export class L1DataTransportService extends BaseService<L1DataTransportServiceOptions> {
  constructor(options: L1DataTransportServiceOptions) {
    super('L1_Data_Transport_Service', options, optionSettings)
  }

  private state: {
    db: LevelUp
    l1IngestionService?: L1IngestionService
    l2IngestionService?: L2IngestionService
    l1TransportServer: L1TransportServer
    metrics: LegacyMetrics
    failureCounter: Counter<string>
  } = {} as any

  protected async _init(): Promise<void> {
    this.logger.info('Initializing L1 Data Transport Service...')

    this.state.db = level(this.options.dbPath)
    await this.state.db.open()

    // BSS HF1 activates at block 0 if not specified.
    const bssHf1Index = BSS_HF1_INDEX[this.options.l2ChainId] || 0
    this.logger.info(`L2 chain ID is: ${this.options.l2ChainId}`)
    this.logger.info(`BSS HF1 will activate at: ${bssHf1Index}`)

    this.state.metrics = new LegacyMetrics({
      labels: {
        environment: this.options.nodeEnv,
        network: this.options.ethNetworkName,
        release: this.options.release,
        service: this.name,
      }
    })

    this.state.metrics.client.collectDefaultMetrics({
      prefix: 'data_transport_layer_'
    })

    this.state.failureCounter = new this.state.metrics.client.Counter({
      name: 'data_transport_layer_main_service_failures',
      help: 'Counts the number of times that the main service fails',
      registers: [this.state.metrics.registry],
    })

    this.state.l1TransportServer = new L1TransportServer({
      ...this.options,
      metrics: this.state.metrics,
      db: this.state.db,
    })

    // Optionally enable sync from L1.
    if (this.options.syncFromL1) {
      this.state.l1IngestionService = new L1IngestionService({
        ...this.options,
        metrics: this.state.metrics,
        db: this.state.db,
      })
    }

    // Optionally enable sync from L2.
    if (this.options.syncFromL2) {
      this.state.l2IngestionService = new L2IngestionService({
        ...(this.options as any), // TODO: Correct thing to do here is to assert this type.
        metrics: this.state.metrics,
        db: this.state.db,
      })
    }

    await this.state.l1TransportServer.init()

    if (this.options.syncFromL1) {
      await this.state.l1IngestionService.init()
    }

    if (this.options.syncFromL2) {
      await this.state.l2IngestionService.init()
    }
  }

  protected async _start(): Promise<void> {
    try {
      await Promise.all([
        this.state.l1TransportServer.start(),
        this.options.syncFromL1 ? this.state.l1IngestionService.start() : null,
        this.options.syncFromL2 ? this.state.l2IngestionService.start() : null,
      ])
    } catch (e) {
      this.state.failureCounter.inc()
      throw e
    }
  }

  protected async _stop(): Promise<void> {
    try {
      await Promise.all([
        this.state.l1TransportServer.stop(),
        this.options.syncFromL1 ? this.state.l1IngestionService.stop() : null,
        this.options.syncFromL2 ? this.state.l2IngestionService.stop() : null,
      ])

      await this.state.db.close()
    } catch (e) {
      this.state.failureCounter.inc()
      throw e
    }
  }
}
