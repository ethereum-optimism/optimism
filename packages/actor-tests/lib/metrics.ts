import fs from 'fs'
import http from 'http'
import url from 'url'

import client from 'prom-client'

export const metricsRegistry = new client.Registry()

const metricName = (name: string) => {
  return `actor_${name}`
}

export const successfulBenchRunsTotal = new client.Counter({
  name: metricName('successful_bench_runs_total'),
  help: 'Count of total successful bench runs.',
  labelNames: ['actor_name', 'bench_name', 'worker_id'] as const,
  registers: [metricsRegistry],
})

export const failedBenchRunsTotal = new client.Counter({
  name: metricName('failed_bench_runs_total'),
  help: 'Count of total failed bench runs.',
  labelNames: ['actor_name', 'bench_name', 'worker_id'] as const,
  registers: [metricsRegistry],
})

export const benchDurationsSummary = new client.Summary({
  name: metricName('step_durations_ms_summary'),
  help: 'Summary of successful bench durations.',
  percentiles: [0.5, 0.9, 0.95, 0.99],
  labelNames: ['actor_name', 'bench_name'] as const,
  registers: [metricsRegistry],
})

export const successfulActorRunsTotal = new client.Counter({
  name: metricName('successful_actor_runs_total'),
  help: 'Count of total successful actor runs.',
  labelNames: ['actor_name'] as const,
  registers: [metricsRegistry],
})

export const failedActorRunsTotal = new client.Counter({
  name: metricName('failed_actor_runs_total'),
  help: 'Count of total failed actor runs.',
  labelNames: ['actor_name'] as const,
  registers: [metricsRegistry],
})

export const sanitizeForMetrics = (input: string) => {
  return input.toLowerCase().replace(/ /gi, '_')
}

export const dumpMetrics = async (filename: string) => {
  const metrics = await metricsRegistry.metrics()
  await fs.promises.writeFile(filename, metrics, {
    flag: 'w+',
  })
}

export const serveMetrics = (port: number) => {
  const server = http.createServer(async (req, res) => {
    const route = url.parse(req.url).pathname
    if (route !== '/metrics') {
      res.writeHead(404)
      res.end()
      return
    }

    res.setHeader('Content-Type', metricsRegistry.contentType)
    res.end(await metricsRegistry.metrics())
  })
  server.listen(port)
}
