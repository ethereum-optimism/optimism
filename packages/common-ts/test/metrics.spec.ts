import request from 'supertest'
// Setup
import chai = require('chai')
const expect = chai.expect

import { Logger, LegacyMetrics, createMetricsServer } from '../src'

describe('Metrics', () => {
  it('should serve metrics', async () => {
    const metrics = new LegacyMetrics({
      prefix: 'test_metrics',
    })
    const registry = metrics.registry
    const logger = new Logger({ name: 'test_logger' })

    const server = await createMetricsServer({
      logger,
      registry,
      port: 42069,
    })

    try {
      // Create two metrics for testing
      const counter = new metrics.client.Counter({
        name: 'counter',
        help: 'counter help',
        registers: [registry],
      })
      const gauge = new metrics.client.Gauge({
        name: 'gauge',
        help: 'gauge help',
        registers: [registry],
      })

      counter.inc()
      counter.inc()
      gauge.set(100)

      // Verify that the registered metrics are served at `/`
      const response = await request(server).get('/metrics').send()
      expect(response.status).eq(200)
      expect(response.text).match(/counter 2/)
      expect(response.text).match(/gauge 100/)
    } finally {
      server.close()
      registry.clear()
    }
  })
})
