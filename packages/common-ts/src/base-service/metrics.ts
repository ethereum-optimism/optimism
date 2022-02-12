import {
  Gauge as PGauge,
  Counter as PCounter,
  Histogram as PHistogram,
  Summary as PSummary,
} from 'prom-client'

export class Gauge extends PGauge<string> {}
export class Counter extends PCounter<string> {}
export class Histogram extends PHistogram<string> {}
export class Summary extends PSummary<string> {}
export type Metric = Gauge | Counter | Histogram | Summary
