import express from 'express'
import { Server } from 'net'
import promBundle from 'express-prom-bundle'
import { Gauge } from 'prom-client'
import { providers } from 'ethers'
import { Metrics, Logger } from '@eth-optimism/common-ts'
import { injectL2Context, sleep } from '@eth-optimism/core-utils'

import { binarySearchForMismatch } from './helpers'

export interface HealthcheckServerOptions {
  network: string
  gethRelease: string
  sequencerRpcProvider: string
  replicaRpcProvider: string
  logger: Logger
}

export interface ReplicaMetrics {
  lastMatchingStateRootHeight: Gauge<string>
  replicaHeight: Gauge<string>
  sequencerHeight: Gauge<string>
}

export class HealthcheckServer {
  protected options: HealthcheckServerOptions
  protected app: express.Express
  protected logger: Logger
  protected metrics: ReplicaMetrics
  server: Server

  constructor(options: HealthcheckServerOptions) {
    this.options = options
    this.app = express()
    this.logger = options.logger
  }

  init = () => {
    this.metrics = this.initMetrics()
    this.server = this.initServer()
  }

  initMetrics = (): ReplicaMetrics => {
    const metrics = new Metrics({
      labels: {
        network: this.options.network,
        gethRelease: this.options.gethRelease,
      },
    })
    const metricsMiddleware = promBundle({
      includeMethod: true,
      includePath: true,
    })
    this.app.use(metricsMiddleware)

    return {
      lastMatchingStateRootHeight: new metrics.client.Gauge({
        name: 'replica_health_last_matching_state_root_height',
        help: 'Height of last matching state root of replica',
        registers: [metrics.registry],
      }),
      replicaHeight: new metrics.client.Gauge({
        name: 'replica_health_height',
        help: 'Block number of the latest block from the replica',
        registers: [metrics.registry],
      }),
      sequencerHeight: new metrics.client.Gauge({
        name: 'replica_health_sequencer_height',
        help: 'Block number of the latest block from the sequencer',
        registers: [metrics.registry],
      }),
    }
  }

  initServer = (): Server => {
    this.app.get('/', (req, res) => {
      res.send(`
        <head><title>Replica healthcheck</title></head>
        <body>
        <h1>Replica healthcheck</h1>
        <p><a href="/metrics">Metrics</a></p>
        </body>
        </html>
      `)
    })

    const server = this.app.listen(3000, () => {
      this.logger.info('Listening on port 3000')
    })

    return server
  }

  runSyncCheck = async () => {
    const sequencerProvider = injectL2Context(
      new providers.JsonRpcProvider(this.options.sequencerRpcProvider)
    )
    const replicaProvider = injectL2Context(
      new providers.JsonRpcBatchProvider(this.options.replicaRpcProvider)
    )

    // Continuously loop while replica runs
    while (true) {
      let replicaLatest = (await replicaProvider.getBlock('latest')) as any
      const sequencerCorresponding = (await sequencerProvider.getBlock(
        replicaLatest.number
      )) as any

      if (replicaLatest.stateRoot !== sequencerCorresponding.stateRoot) {
        this.logger.error(
          'Latest replica state root is mismatched from sequencer'
        )
        const firstMismatch = await binarySearchForMismatch(
          sequencerProvider,
          replicaProvider,
          replicaLatest.number,
          this.logger
        )
        this.logger.error('First state root mismatch found', {
          blockNumber: firstMismatch,
        })
        this.metrics.lastMatchingStateRootHeight.set(firstMismatch)

        throw new Error('Replica state root mismatched')
      }

      this.logger.info('State roots matching', {
        blockNumber: replicaLatest.number,
      })
      this.metrics.lastMatchingStateRootHeight.set(replicaLatest.number)

      replicaLatest = await replicaProvider.getBlock('latest')
      const sequencerLatest = await sequencerProvider.getBlock('latest')
      this.logger.info('Syncing from sequencer', {
        sequencerHeight: sequencerLatest.number,
        replicaHeight: replicaLatest.number,
        heightDifference: sequencerLatest.number - replicaLatest.number,
      })

      this.metrics.replicaHeight.set(replicaLatest.number)
      this.metrics.sequencerHeight.set(sequencerLatest.number)
      // Fetch next block and sleep if not new
      while (replicaLatest.number === sequencerCorresponding.number) {
        this.logger.info(
          'Replica caught up with sequencer, waiting for next block'
        )
        await sleep(1_000)
        replicaLatest = await replicaProvider.getBlock('latest')
      }
    }
  }
}
