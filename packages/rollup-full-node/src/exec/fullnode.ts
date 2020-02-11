/* External Imports */
import {
  getLogger,
  JsonRpcRequest,
  JsonRpcResponse,
  logError,
  Logger,
} from '@eth-optimism/core-utils'
import cors = require('cors')
import * as fs from 'fs'
import { config, parse } from 'dotenv'

/* Internal Imports */
import {
  FullnodeRpcServer,
  deployOvmContract,
  DefaultWeb3Handler,
} from '../app'
import { resolve } from 'path'

const log: Logger = getLogger('rollup-fullnode')

const supportedMethodsKey: string = 'supportedJsonRpcMethods'

const getConfigFilePath = (filename: string): string => {
  return resolve(__dirname, `../../../config/${filename}`)
}

const getConfig = (): {} => {
  // Starting from build/src/exec/
  if (!fs.existsSync(getConfigFilePath('.env'))) {
    log.debug(`No override config found at 'config/.env'.`)
  } else {
    config({ path: getConfigFilePath('.env') })
  }

  const defaultFilepath: string = getConfigFilePath('default.json')
  if (!fs.existsSync(defaultFilepath)) {
    log.error(`No 'default.json' config file found in /config dir`)
    process.exit(1)
  }

  let defaultConfig: {}
  try {
    defaultConfig = JSON.parse(
      fs.readFileSync(defaultFilepath, { encoding: 'utf8' })
    )
  } catch (e) {
    log.error("Invalid JSON in 'config/default.json' config file.")
    process.exit(1)
  }

  return defaultConfig
}

const runFullnode = async (): Promise<void> => {
  // TODO Get these from config
  const host = '0.0.0.0'
  const port = 8545

  const fullnodeHandler = await DefaultWeb3Handler.create()
  const fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

  fullnodeRpcServer.listen()

  const baseUrl = `http://${host}:${port}`

  log.info(`Listening on ${host}:${port}`)
}

// Start Fullnode
runFullnode()
