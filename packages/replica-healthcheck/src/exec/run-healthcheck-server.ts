import * as dotenv from 'dotenv'

import { HealthcheckServer, readConfig } from '..'
;(async () => {
  dotenv.config()

  const healthcheckServer = new HealthcheckServer(readConfig())

  healthcheckServer.init()
  await healthcheckServer.runSyncCheck()
})().catch((err) => {
  console.log(err)
  process.exit(1)
})
