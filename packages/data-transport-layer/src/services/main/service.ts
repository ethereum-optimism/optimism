/* Imports: External */
import { BaseService } from '@eth-optimism/common-ts'
import { LevelUp } from 'levelup'
import level from 'level'

/* Imports: Internal */
import { L1IngestionService } from '../l1-ingestion/service'
import { L1TransportServer } from '../server/service'
import { validators } from '../../utils'
import { L2IngestionService } from '../l2-ingestion/service'

export interface L1DataTransportServiceOptions {
  nodeEnv: string
  ethNetworkName?: 'mainnet' | 'kovan' | 'goerli'
  release: string
  addressManager: string
  confirmations: number
  dangerouslyCatchAllErrors?: boolean
  hostname: string
  l1RpcProvider: string
  l2ChainId: number
  l2RpcProvider: string
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
  enableMetrics?: boolean
  defaultBackend: string
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
  } = {} as any

  protected async _init(): Promise<void> {
    this.logger.info('Initializing L1 Data Transport Service...')

    this.state.db = level(this.options.dbPath)
    await this.state.db.open()

    this.state.l1TransportServer = new L1TransportServer({
      ...this.options,
      db: this.state.db,
    })

    // Optionally enable sync from L1.
    if (this.options.syncFromL1) {
      this.state.l1IngestionService = new L1IngestionService({
        ...this.options,
        db: this.state.db,
      })
    }

    // Optionally enable sync from L2.
    if (this.options.syncFromL2) {
      this.state.l2IngestionService = new L2IngestionService({
        ...(this.options as any), // TODO: Correct thing to do here is to assert this type.
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
    await Promise.all([
      this.state.l1TransportServer.start(),
      this.options.syncFromL1 ? this.state.l1IngestionService.start() : null,
      this.options.syncFromL2 ? this.state.l2IngestionService.start() : null,
    ])
  }

  protected async _stop(): Promise<void> {
    await Promise.all([
      this.state.l1TransportServer.stop(),
      this.options.syncFromL1 ? this.state.l1IngestionService.stop() : null,
      this.options.syncFromL2 ? this.state.l2IngestionService.stop() : null,
    ])

    await this.state.db.close()
  }
}
