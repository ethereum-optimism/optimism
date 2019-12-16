/* External Imports */
import { EVMOpcode, Opcode, Address } from '@pigi/rollup-core'
import { getLogger, logError, hexStrToBuf, remove0x } from '@pigi/core-utils'

import * as fs from 'fs'
import { config, parse } from 'dotenv'
import { resolve } from 'path'

/* Internal Imports */
import { OpcodeWhitelist } from '../../types/transpiler'
import { OpcodeWhitelistImpl, OpcodeReplacementsImpl } from '../'

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
 * Gets the specified state manager address from configuration.
 *
 * @returns The hex string of the state manager address if successful, undefined if not.
 */
function getStateManagerAddress(defaultConfig: {}): string | undefined {
  const stateManagerAddress: string =
    process.env.STATE_MANAGER_ADDRESS || defaultConfig['STATE_MANAGER_ADDRESS']
  if (!stateManagerAddress) {
    log.error(
      `No state manager address specified. Please configure STATE_MANAGER_ADDRESS in either 'config/.env.default' or as an environment variable.`
    )
    return undefined
  }

  log.info(
    `Got the following state manager address from config: [${stateManagerAddress}].`
  )

  if (
    !(
      stateManagerAddress.length === 42 &&
      stateManagerAddress.slice(0, 2) === '0x'
    )
  ) {
    log.error(
      `[${stateManagerAddress}] does not appear to be a valid hex string address.`
    )
    return undefined
  }

  return stateManagerAddress
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

  const stateManagerAddress: string = getStateManagerAddress(defaultConfig)
  log.info(`SM address is : ${stateManagerAddress.toString()}`)
  const opcodeReplacements = new OpcodeReplacementsImpl(stateManagerAddress)
  // TODO: Instantiate all of the things and call transpiler.transpile()
}

transpile()
