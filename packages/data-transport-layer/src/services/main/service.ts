/* Imports: External */
import { BaseService, Logger, Metrics } from '@eth-optimism/common-ts'
import { LevelUp } from 'levelup'
import level from 'level'

/* Imports: Internal */
import { L1IngestionService } from '../l1-ingestion/service'
import { L1TransportServer } from '../server/service'
import { sleep, validators } from '../../utils'
import { L2IngestionService } from '../l2-ingestion/service'
import { Counter } from 'prom-client'

import express, { Request, Response } from 'express'
import bodyParser from 'body-parser'
import cors from 'cors'

export interface L1DataTransportServiceOptions {
  nodeEnv: string
  ethNetworkName?: 'mainnet' | 'kovan' | 'goerli'
  release: string
  cfgAddressManager: string
  confirmations: number
  dangerouslyCatchAllErrors?: boolean
  hostname: string
  l1RpcProvider: string
  l2ChainId: number
  l2RpcProvider: string
  metrics?: Metrics
  dbPath: string
  logsPerPollingInterval: number
  pollingInterval: number
  port: number
  arPort: number
  syncFromL1?: boolean
  syncFromL2?: boolean
  transactionsPerPollingInterval: number
  legacySequencerCompatibility: boolean
  useSentry?: boolean
  sentryDsn?: string
  sentryTraceRate?: number
  defaultBackend: string
  l1GasPriceBackend: string
  ctcDeploymentHeight?: number
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
    addressManager: string
    db: LevelUp
    l1IngestionService?: L1IngestionService
    l2IngestionService?: L2IngestionService
    l1TransportServer: L1TransportServer
    metrics: Metrics
    failureCounter: Counter<string>
    addressRegistry: express.Express
    arServer: any
  } = {} as any

  protected async _init(): Promise<void> {
    this.logger.info('Initializing L1 Data Transport Service...')

    this.state.db = level(this.options.dbPath)
    await this.state.db.open()

    this.state.metrics = new Metrics({
      labels: {
        environment: this.options.nodeEnv,
        network: this.options.ethNetworkName,
        release: this.options.release,
        service: this.name,
      }
    })
    this.state.failureCounter = new this.state.metrics.client.Counter({
      name: 'data_transport_layer_main_service_failures',
      help: 'Counts the number of times that the main service fails',
      registers: [this.state.metrics.registry],
    })

    this.state.addressRegistry = express()
    this.state.addressRegistry.use(cors())
    this.state.addressRegistry.use(bodyParser.json())

    // Could refactor this to reduce code duplication. For now the goal is just to
    // have something which works reliably.

    this.state.addressRegistry['get']("/addresses.json", async (req, res) => {
      try {
        let aList
        try {
          aList = JSON.parse(await this.state.db.get("address-list"))
        } catch(e) {
          if (e.notFound) {
            this.logger.warn("Address Registry is not yet ready to serve addresses (db notFound)")
            return res.status(503).json({error: "Address Registry is not yet populated"})
          } else { throw e }
        }

        return res.json(aList)
      } catch (e) {
        return res.status(400).json({
          error: e.toString(),
        })
      }
    })
    this.state.addressRegistry['get']("/omgx-addr.json", async (req, res) => {
      try {
        let aList
        try {
          aList = JSON.parse(await this.state.db.get("omgx-addr"))
        } catch(e) {
          if (e.notFound) {
            this.logger.warn("Address Registry is not yet ready to serve OMGX addresses (db notFound)")
            return res.status(503).json({error: "Address Registry is not yet populated"})
          } else { throw e }
        }

        return res.json(aList)
      } catch (e) {
        return res.status(400).json({
          error: e.toString(),
        })
      }
    })
    this.state.addressRegistry['put']("/addresses.json", async (req, res) => {
      try {
        const rb = req.body

        this.logger.info("addressRegistry PUT request for base addresses", {rb})

        let addrList = {}

        /* Future - if there's an existing list, compare it with the new one and reject
           any attempts to change certain critical addresses. For now, only cares about
           AddressManager since that's used directly by dtl */

        try {
          const _addrList = await this.state.db.get("address-list")
          addrList = JSON.parse(_addrList)

          if (rb['AddressManager'] && addrList['AddressManager'] && rb['AddressManager'] !== addrList['AddressManager']) {
            this.logger.error("Can't overwrite saved addressManager value", { old: addrList['AddressManager'], new:rb['AddressManager']})
            return res.status(400).json({error:"Can't overwrite saved AddressManager value"})
          }
        } catch(e) {
          if (e.notFound) {
            this.logger.info("No previous address list was found")
          } else { throw e; }
        }

        this.logger.info("MMDBG Will store", rb)
        await this.state.db.put("address-list", JSON.stringify(rb))
        this.logger.info("MMDBG Stored")
        return res.sendStatus(201).end()
      } catch (e) {
        return res.status(400).json({
          error: e.toString(),
        })
      }
    })
    this.state.addressRegistry['put']("/omgx-addr.json", async (req, res) => {
      try {
        const rb = req.body

        this.logger.info("addressRegistry PUT request for OMGX addresses", {rb})

        // As with the base list, we could add future restrictions on changing
        // certain critical addresses. For now we allow anything.

        this.logger.info("MMDBG Will store", rb)
        await this.state.db.put("omgx-addr", JSON.stringify(rb))
        this.logger.info("MMDBG Stored")
        return res.sendStatus(201).end()
      } catch (e) {
        return res.status(400).json({
          error: e.toString(),
        })
      }
    })

    this.state.addressRegistry['get']("/state-dump.latest.json", async (req, res) => {
      try {
        //const sd = await this.state.db.get("state-dump")
        //return res.send()
        return res.sendFile("/opt/optimism/packages/data-transport-layer/state-dumps/state-dump.latest.json")

      } catch (e) {
        return res.status(400).json({
          error: e.toString(),
        })
      }
    })
    this.state.arServer = this.state.addressRegistry.listen(this.options.arPort,this.options.hostname)
    this.logger.info("MMDBG addressRegistry server listening", {hostname:this.options.hostname, port:this.options.arPort})

    if (this.options.cfgAddressManager) {
      this.logger.warn("Using legacy cfgAddressManager address")
      this.state.addressManager = this.options.cfgAddressManager
    }

    do {
     let addrList
     try {
        addrList = JSON.parse(await this.state.db.get("address-list"))
        this.state.addressManager = addrList['AddressManager']
      } catch(e) {
        if (! e.notFound) { throw e }
      }

      if (! this.state.addressManager) {
        this.logger.info("Waiting for initial ADDRESS_MANAGER address")
        await sleep(5000)
      }
    } while (! this.state.addressManager)

    this.logger.info("addressManager set, continuing with startup", {addr:this.state.addressManager})

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
        addressManager: this.state.addressManager
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
