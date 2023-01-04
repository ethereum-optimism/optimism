import {
  Gauge as PGauge,
  Counter as PCounter,
  Histogram as PHistogram,
  Summary as PSummary,
} from 'prom-client'

import { OptionsSpec, getPublicOptions } from './options'

// Prometheus metrics re-exported.
export class Gauge extends PGauge<string> {}
export class Counter extends PCounter<string> {}
export class Histogram extends PHistogram<string> {}
export class Summary extends PSummary<string> {}
export type Metric = Gauge | Counter | Histogram | Summary

/**
 * Metrics that are available for a given service.
 */
export type Metrics = Record<any, Metric>

/**
 * Specification for metrics.
 */
export type MetricsSpec<TMetrics extends Metrics> = {
  [P in keyof Required<TMetrics>]: {
    type: new (configuration: any) => TMetrics[P]
    desc: string
    labels?: string[]
  }
}

/**
 * Standard metrics that are always available.
 */
export type StandardMetrics = {
  metadata: Gauge
  unhandledErrors: Counter
}

/**
 * Generates a standard metrics specification. Needs to be a function because the labels for
 * service metadata are dynamic dependent on the list of given options.
 *
 * @param options Options to include in the service metadata.
 * @returns Metrics specification.
 */
export const makeStdMetricsSpec = (
  optionsSpec: OptionsSpec<any>
): MetricsSpec<StandardMetrics> => {
  return {
    // Users cannot set these options.
    metadata: {
      type: Gauge,
      desc: 'Service metadata',
      labels: ['name', 'version'].concat(getPublicOptions(optionsSpec)),
    },
    unhandledErrors: {
      type: Counter,
      desc: 'Unhandled errors',
    },
  }
}
