import { Server } from 'net'

import express from 'express'
import promBundle from 'express-prom-bundle'
import { Gauge, Histogram } from 'prom-client'
import cron from 'node-cron'
import { providers, Wallet } from 'ethers'
import { Metrics, Logger } from '@eth-optimism/common-ts'
import { sleep } from '@eth-optimism/core-utils'
import { asL2Provider } from '@eth-optimism/sdk'

import { binarySearchForMismatch } from './helpers'

export interface HealthcheckServerOptions {
  network: string
  gethRelease: string
  sequencerRpcProvider: string
  replicaRpcProvider: string
  checkTxWriteLatency: boolean
  txWriteOptions?: TxWriteOptions
  logger: Logger
}

export interface TxWriteOptions {
  wallet1PrivateKey: string
  wallet2PrivateKey: string
}

export interface ReplicaMetrics {
  lastMatchingStateRootHeight: Gauge<string>
  replicaHeight: Gauge<string>
  sequencerHeight: Gauge<string>
  txWriteLatencyMs: Histogram<string>
}

export class HealthcheckServer {
  protected options: HealthcheckServerOptions
  protected app: express.Express
  protected logger: Logger
  protected metrics: ReplicaMetrics
  protected replicaProvider: providers.StaticJsonRpcProvider
  server: Server

  constructor(options: HealthcheckServerOptions) {
    this.options = options
    this.app = express()
    this.logger = options.logger
  }

  init = () => {
    this.metrics = this.initMetrics()
    this.server = this.initServer()
    this.replicaProvider = asL2Provider(
      new providers.StaticJsonRpcProvider({
        url: this.options.replicaRpcProvider,
        headers: { 'User-Agent': 'replica-healthcheck' },
      })
    )
    if (this.options.checkTxWriteLatency) {
      this.initTxLatencyCheck()
    }
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
      txWriteLatencyMs: new metrics.client.Histogram({
        name: 'tx_write_latency_in_ms',
        help: 'The latency of sending a write transaction through a replica in ms',
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

  initTxLatencyCheck = () => {
    // Check latency for every Monday
    cron.schedule('0 0 * * 1', this.runTxLatencyCheck)
  }

  runTxLatencyCheck = async () => {
    const wallet1 = new Wallet(
      this.options.txWriteOptions.wallet1PrivateKey,
      this.replicaProvider
    )
    const wallet2 = new Wallet(
      this.options.txWriteOptions.wallet2PrivateKey,
      this.replicaProvider
    )

    // Send funds between the 2 addresses
    try {
      const res1 = await this.getLatencyForSend(wallet1, wallet2)
      this.logger.info('Sent transaction from wallet1 to wallet2', {
        latencyMs: res1.latencyMs,
        status: res1.status,
      })

      const res2 = await this.getLatencyForSend(wallet2, wallet2)
      this.logger.info('Sent transaction from wallet2 to wallet1', {
        latencyMs: res2.latencyMs,
        status: res2.status,
      })
    } catch (err) {
      this.logger.error('Failed to get tx write latency', {
        message: err.toString(),
        stack: err.stack,
        code: err.code,
        wallet1: wallet1.address,
        wallet2: wallet2.address,
      })
    }
  }

  getLatencyForSend = async (
    from: Wallet,
    to: Wallet
  ): Promise<{
    latencyMs: number
    status: number
  }> => {
    const fromBal = await from.getBalance()
    if (fromBal.isZero()) {
      throw new Error('Wallet balance is zero, cannot make test transaction')
    }

    const startTime = new Date()
    const tx = await from.sendTransaction({
      to: to.address,
      value: fromBal.div(2), // send half
    })
    const { status } = await tx.wait()
    const endTime = new Date()
    const latencyMs = endTime.getTime() - startTime.getTime()
    this.metrics.txWriteLatencyMs.observe(latencyMs)
    return { latencyMs, status }
  }

  runSyncCheck = async () => {
    const sequencerProvider = asL2Provider(
      new providers.StaticJsonRpcProvider({
        url: this.options.sequencerRpcProvider,
        headers: { 'User-Agent': 'replica-healthcheck' },
      })
    )

    // Continuously loop while replica runs
    while (true) {
      let replicaLatest = (await this.replicaProvider.getBlock('latest')) as any
      const sequencerCorresponding = (await sequencerProvider.getBlock(
        replicaLatest.number
      )) as any

      if (replicaLatest.stateRoot !== sequencerCorresponding.stateRoot) {
        this.logger.error(
          'Latest replica state root is mismatched from sequencer'
        )
        const firstMismatch = await binarySearchForMismatch(
          sequencerProvider,
          this.replicaProvider,
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

      replicaLatest = await this.replicaProvider.getBlock('latest')
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
        replicaLatest = await this.replicaProvider.getBlock('latest')
      }
    }
  }
}
