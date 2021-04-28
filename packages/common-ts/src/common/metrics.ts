import prometheus, {
  collectDefaultMetrics,
  DefaultMetricsCollectorConfiguration,
  Registry,
} from 'prom-client'

export interface MetricsOptions {
  prefix: string
  labels?: Object
}

export class Metrics {
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
