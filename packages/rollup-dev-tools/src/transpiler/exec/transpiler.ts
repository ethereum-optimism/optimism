/* External Imports */
import { EVMOpcode, Opcode } from '@pigi/rollup-core'
import { getLogger, logError } from '@pigi/core-utils'

import * as fs from 'fs'
import { config, parse } from 'dotenv'
import { resolve } from 'path'

/* Internal Imports */
import { OpcodeWhitelist } from '../../types/transpiler'
import { OpcodeWhitelistImpl } from '../opcode-whitelist'

const log = getLogger('transpiler')

/**
 * Creates an OpcodeWhitelist from configuration.
 *
 * @returns The constructed OpcodeWhitelist if successful, undefined if not.
 */
function getOpcodeWhitelist(defaultConfig: {}): OpcodeWhitelist | undefined {
  const configuredWhitelist: string =
    process.env.OPCODE_WHITELIST || defaultConfig['OPCODE_WHITELIST']
  if (!configuredWhitelist) {
    log.error(
      `No op codes whitelisted. Please configure OPCODE_WHITELIST in either 'config/.env.default' or as an environment variable.`
    )
    return undefined
  }

  log.info(`Parsing whitelisted op codes: [${configuredWhitelist}].`)

  const whitelistOpcodeStrings: string[] = configuredWhitelist
    .split(',')
    .map((x) => x.trim().toUpperCase())
  const whiteListedOpCodes: EVMOpcode[] = []
  const invalidOpcodes: string[] = []
  for (const opString of whitelistOpcodeStrings) {
    const opcode: EVMOpcode = Opcode.parseByName(opString)
    if (!opcode) {
      invalidOpcodes.push(opString)
    } else {
      whiteListedOpCodes.push(opcode)
    }
  }

  if (!!invalidOpcodes.length) {
    log.error(
      `The following configured opcodes are not valid opcodes: ${invalidOpcodes.join(
        ','
      )}`
    )
    return undefined
  }

  if (!whiteListedOpCodes.length) {
    log.error(
      `There are no configured whitelisted opcodes. Transpilation cannot work without supporting some opcodes`
    )
    return undefined
  }

  return new OpcodeWhitelistImpl(whiteListedOpCodes)
}

/**
 * Helper function for getting config file paths to avoid base path duplication.
 *
 * @param filename The config filename without path.
 * @returns The full config file path.
 */
function getConfigFilePath(filename: string): string {
  return resolve(__dirname, `../../../../config/${filename}`)
}

/**
 * Loads environment variables and returns default config.
 *
 * @returns The default configuration key-value pairs
 */
async function loadEnvironment(): Promise<{}> {
  // Starting from build/src/
  if (!(await fs.existsSync(getConfigFilePath('.env')))) {
    log.debug(`No override config found at 'config/.env'.`)
  } else {
    config({ path: getConfigFilePath('.env') })
  }

  let configuration: {} = {}

  if (!(await fs.existsSync(getConfigFilePath('.env.default')))) {
    log.info(
      `No default config found at 'config/.env.default'. This is probably bad, but transpilation will be attempted anyway.`
    )
  } else {
    try {
      configuration = parse(fs.readFileSync(getConfigFilePath('.env.default')))
    } catch (e) {
      logError(log, `Config file at '.env.default' not formatted properly.`, e)
      return undefined
    }
  }

  return configuration
}

/**
 * Entrypoint for transpilation
 */
async function transpile() {
  const defaultConfig = await loadEnvironment()
  if (defaultConfig === undefined) {
    return
  }

  const opcodeWhitelist: OpcodeWhitelist = getOpcodeWhitelist(defaultConfig)
  if (!opcodeWhitelist) {
    return
  }

  // TODO: Instantiate all of the things and call transpiler.transpile()
}

transpile()
