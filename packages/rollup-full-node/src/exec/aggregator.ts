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
import { Aggregator } from '../types'
import { AggregatorRpcServer } from '../app/aggregator-rpc-server'
import { resolve } from 'path'

const log: Logger = getLogger('aggregator-exec')

class DummyAggregator implements Aggregator {
  public async handleRequest(
    request: JsonRpcRequest
  ): Promise<JsonRpcResponse> {
    return {
      id: request.id,
      jsonrpc: request.jsonrpc,
      result: 'Not Implemented =|',
    }
  }
}

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

/**
 * Gets the set of supported methods from config.
 *
 * @returns The supported methods for the Aggregator
 */
const getSupportedMethods = (parsedConfig: {}): Set<string> => {
  const override: string = process.env[supportedMethodsKey]
  if (override) {
    let arr: string[]
    try {
      arr = override.split(',').map((x) => x.trim())
    } catch (e) {
      log.error(
        `Override for ${supportedMethodsKey} configured but in the wrong format. Must be a comma-separated string.`
      )
      process.exit(1)
    }
    return new Set<string>(arr)
  }

  if (supportedMethodsKey in config) {
    log.error(`No ${supportedMethodsKey} defined in config.`)
    process.exit(1)
  }

  return new Set<string>(parsedConfig[supportedMethodsKey])
}

export const runAggregator = async (): Promise<void> => {
  // TODO: Replace with actual Aggregator when wired up.
  const dummyAggregator: Aggregator = new DummyAggregator()

  // TODO: get these from config
  const host = '0.0.0.0'
  const port = 3000

  const parsedConfig = getConfig()
  const supportedMethods: Set<string> = getSupportedMethods(parsedConfig)
  const server: AggregatorRpcServer = new AggregatorRpcServer(
    supportedMethods,
    dummyAggregator,
    host,
    port,
    [cors]
  )

  server.listen()

  log.info(`Listening on ${host}:${port}`)
}
