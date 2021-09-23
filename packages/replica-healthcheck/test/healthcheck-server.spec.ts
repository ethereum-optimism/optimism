import request from 'supertest'
// Setup
import chai = require('chai')
const expect = chai.expect
import { Logger } from '@eth-optimism/common-ts'

import { HealthcheckServer } from '../src/healthcheck-server'

describe('HealthcheckServer', () => {
  it('shoud serve correct metrics', async () => {
    const logger = new Logger({ name: 'test_logger' })
    const healthcheckServer = new HealthcheckServer({
      network: 'kovan',
      gethRelease: '0.4.20',
      sequencerRpcProvider: 'http://sequencer.io',
      replicaRpcProvider: 'http://replica.io',
      logger,
    })

    try {
      await healthcheckServer.init()
      // Verify that the registered metrics are served at `/`
      const response = await request(healthcheckServer.server)
        .get('/metrics')
        .send()
      expect(response.status).eq(200)
      expect(response.text).match(/replica_health_height gauge/)
    } finally {
      healthcheckServer.server.close()
    }
  })
})
