import { Server } from 'net'

import prometheus, {
  collectDefaultMetrics,
  DefaultMetricsCollectorConfiguration,
  Registry,
} from 'prom-client'
import express from 'express'

import { Logger } from './logger'

export interface MetricsOptions {
  prefix?: string
  labels?: Object
}

export class LegacyMetrics {
  options: MetricsOptions
  client: typeof prometheus
  registry: Registry

  constructor(options: MetricsOptions) {
    this.options = options

    const metricsOptions: DefaultMetricsCollectorConfiguration = {
      prefix: options.prefix,
      labels: options.labels,
    }

    this.client = prometheus
    this.registry = prometheus.register

    // Collect default metrics (event loop lag, memory, file descriptors etc.)
    collectDefaultMetrics(metricsOptions)
  }
}

export interface MetricsServerOptions {
  logger: Logger
  registry: Registry
  port?: number
  route?: string
  hostname?: string
}

export const createMetricsServer = async (
  options: MetricsServerOptions
): Promise<Server> => {
  const logger = options.logger.child({ component: 'MetricsServer' })

  const app = express()

  const route = options.route || '/metrics'
  app.get(route, async (_, res) => {
    res.status(200).send(await options.registry.metrics())
  })

  const port = options.port || 7300
  const hostname = options.hostname || '0.0.0.0'
  const server = app.listen(port, hostname, () => {
    logger.info('Metrics server started', {
      port,
      hostname,
      route,
    })
  })

  return server
}
