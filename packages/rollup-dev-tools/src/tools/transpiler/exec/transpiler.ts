/* External Imports */
import { EVMOpcode, Opcode, Address, EVMBytecode } from '@pigi/rollup-core'
import {
  getLogger,
  logError,
  isValidHexAddress,
  remove0x,
  bufToHexString,
} from '@pigi/core-utils'

import * as fs from 'fs'
import { config, parse } from 'dotenv'
import { resolve } from 'path'

/* Internal Imports */
import {
  ErroredTranspilation,
  OpcodeWhitelist,
  SuccessfulTranspilation,
  TranspilationResult,
  Transpiler,
} from '../../../types/transpiler'
import {
  OpcodeWhitelistImpl,
  OpcodeReplacerImpl,
  TranspilerImpl,
} from '../index'

const log = getLogger('transpiler')

/**
 * Creates an OpcodeWhitelist from configuration.
 *
 * @returns The constructed OpcodeWhitelist if successful, undefined if not.
 */
function getOpcodeWhitelist(defaultConfig: {}): OpcodeWhitelist | undefined {
  const configuredWhitelist: string[] = process.env.OPCODE_WHITELIST
    ? process.env.OPCODE_WHITELIST.split(',')
    : defaultConfig['OPCODE_WHITELIST']
  if (!configuredWhitelist) {
    console.error(
      `No op codes whitelisted. Please configure OPCODE_WHITELIST in either 'config/default.json' or as an environment variable.`
    )
    return undefined
  }

  log.debug(`Parsing whitelisted op codes: [${configuredWhitelist}].`)

  const whitelistOpcodeStrings: string[] = configuredWhitelist.map((x) =>
    x.trim().toUpperCase()
  )
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
    console.error(
      `The following configured opcodes are not valid opcodes: ${invalidOpcodes.join(
        ','
      )}`
    )
    return undefined
  }

  if (!whiteListedOpCodes.length) {
    console.error(
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
function getStateManagerAddress(defaultConfig: {}): Address {
  const stateManagerAddress: Address =
    process.env.STATE_MANAGER_ADDRESS || defaultConfig['STATE_MANAGER_ADDRESS']
  if (!stateManagerAddress) {
    console.error(
      `No state manager address specified. Please configure STATE_MANAGER_ADDRESS in either 'config/default.json' or as an environment variable.`
    )
    process.exit(1)
  }

  log.debug(
    `Got the following state manager address from config: [${stateManagerAddress}].`
  )

  if (!isValidHexAddress(stateManagerAddress)) {
    console.error(
      `[${stateManagerAddress}] does not appear to be a valid hex string address.`
    )
    process.exit(1)
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
const getConfig = (): {} => {
  // Starting from build/src/transpiler/exec/
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
 * Gets the relative path of a file assuming base path is the `rollup-dev-tools` dir.
 *
 * @param filePath The relative filepath from `rollup-dev-tools`
 * @returns The relative filepath from this built executable
 */
const getRelativePath = (filePath: string): string => {
  return resolve(__dirname, `../../../../${filePath}`)
}

/**
 * Prints the usage of this executable as an error and exits.
 */
const printUsageAndExit = (): void => {
  console.error(
    'Invalid argument(s). Usage: "yarn transpile <inputFilePath> <outputFilePath>" or "yarn transpile <hex string to transpile>"'
  )
  process.exit(1)
}

/**
 * Makes sure the necessary parameters are passed and parse them.
 * @returns an array of [inputFilePath, outputFilePath]
 */
const getParams = (): [Buffer, string | undefined] => {
  if (process.argv.length === 4) {
    // Get the environment and read the appropriate environment file
    const [inputFilePath, outputFilePath] = process.argv
      .slice(process.argv.length - 2)
      .map((x) => getRelativePath(x))

    if (!fs.existsSync(inputFilePath)) {
      console.error(`Input file does not exist at path ${inputFilePath}`)
      printUsageAndExit()
    }

    const inputBytecode: Buffer = fs.readFileSync(inputFilePath)

    return [inputBytecode, outputFilePath]
  }

  if (process.argv.length === 3) {
    try {
      const bytes = Buffer.from(remove0x(process.argv[2]), 'hex')
      return [bytes, undefined]
    } catch (e) {
      console.error(`Argument not a valid hex string: "${process.argv[2]}"`)
      printUsageAndExit()
    }
  }

  printUsageAndExit()
}

const getReplacements = (): Map<EVMOpcode, EVMBytecode> => {
  // TODO: Read in overrides from config
  return new Map<EVMOpcode, EVMBytecode>()
    .set(Opcode.RETURN, [{ opcode: Opcode.AND, consumedBytes: undefined }])
    .set(Opcode.BYTE, [
      { opcode: Opcode.SUB, consumedBytes: undefined },
      { opcode: Opcode.ADD, consumedBytes: undefined },
    ])
}

/**
 * Entrypoint for transpilation
 */
async function transpile() {
  const [inputBytecode, outputFilePath] = getParams()

  const defaultConfig = getConfig()

  const opcodeWhitelist: OpcodeWhitelist = getOpcodeWhitelist(defaultConfig)
  if (!opcodeWhitelist) {
    return
  }

  const stateManagerAddress: Address = getStateManagerAddress(defaultConfig)
  log.debug(`SM address is : ${stateManagerAddress.toString()}`)
  const opcodeReplacer = new OpcodeReplacerImpl(
    stateManagerAddress,
    getReplacements()
  )

  const transpiler: Transpiler = new TranspilerImpl(
    opcodeWhitelist,
    opcodeReplacer
  )

  log.debug(`Transpiling bytecode ${bufToHexString(inputBytecode)}`)

  let result: TranspilationResult
  try {
    result = transpiler.transpile(inputBytecode)
  } catch (e) {
    logError(
      log,
      `Error during transpilation! Input (hex) ${bufToHexString(
        inputBytecode
      )}`,
      e
    )
  }

  if (!result.succeeded) {
    const e: ErroredTranspilation = result as ErroredTranspilation
    console.error(
      `Transpilation Errors: \n\t${e.errors
        .map((x) => `index ${x.index}: ${x.message}`)
        .join('\n\t')}`
    )
  } else {
    const output: SuccessfulTranspilation = result as SuccessfulTranspilation
    if (!!outputFilePath) {
      log.debug(`Transpilation result ${bufToHexString(output.bytecode)}`)
      fs.writeFileSync(outputFilePath, output.bytecode)
    } else {
      console.log(bufToHexString(output.bytecode))
    }
  }
}

transpile()
