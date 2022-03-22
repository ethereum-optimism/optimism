import fetch from 'node-fetch'

import { expect } from './shared/setup'
import { envConfig } from './shared/utils'

describe('Healthcheck Tests', () => {
  before(async function () {
    if (!envConfig.RUN_HEALTHCHECK_TESTS) {
      this.skip()
    }
  })

  // Super simple test, is the metric server up?
  it('should have metrics exposed', async () => {
    const response = await fetch(envConfig.HEALTHCHECK_URL)
    expect(response.status).to.equal(200)
  })
})
