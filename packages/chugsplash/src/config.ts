/* Imports: External */
import * as Handlebars from 'handlebars'
import { ethers } from 'ethers'

/* Imports: Internal */
import { computeStorageSlots, getStorageLayout } from './storage'
import {
  ChugSplashAction,
  ChugSplashActionBundle,
  getChugSplashActionBundle,
} from './actions'
import { getContractArtifact } from './artifacts'

type SolidityVariable =
  | boolean
  | string
  | number
  | Array<SolidityVariable>
  | {
      [name: string]: SolidityVariable
    }

export interface ChugSplashConfig {
  contracts: {
    [name: string]: {
      address: string
      source: string
      variables?: {
        [name: string]: SolidityVariable
      }
    }
  }
}

/**
 * Validates a ChugSplash config file.
 * @param config Config file to validate.
 */
const validateChugSplashConfig = (config: ChugSplashConfig) => {
  if (config.contracts === undefined) {
    throw new Error('contracts field must be defined in ChugSplash config')
  }

  for (const [contractName, contractConfig] of Object.entries(
    config.contracts
  )) {
    // Block people from accidentally using templates in contract names.
    if (contractName.includes('{') || contractName.includes('}')) {
      throw new Error(
        `cannot use template strings in contract names: ${contractName}`
      )
    }

    // Block people from accidentally using templates in contract names.
    if (
      contractConfig.source.includes('{') ||
      contractConfig.source.includes('}')
    ) {
      throw new Error(
        `cannot use template strings in contract source names: ${contractConfig.source}`
      )
    }

    // Make sure addresses are fixed and are actually addresses.
    if (!ethers.utils.isAddress(contractConfig.address)) {
      throw new Error(
        `contract address is not a valid address: ${contractConfig.address}`
      )
    }
  }
}

/**
 * Parses a ChugSplash config file by replacing template values.
 * @param config Unparsed config file to parse.
 * @param env Environment variables to inject into the file.
 * @return Parsed config file with template variables replaced.
 */
export const parseChugSplashConfig = (
  config: ChugSplashConfig,
  env: any = {}
): ChugSplashConfig => {
  validateChugSplashConfig(config)

  const contracts = {}
  for (const [contractName, contractConfig] of Object.entries(
    config.contracts
  )) {
    contracts[contractName] = contractConfig.address
  }

  return JSON.parse(
    Handlebars.compile(JSON.stringify(config))({
      env: new Proxy(env, {
        get: (target, prop) => {
          const val = target[prop]
          if (val === undefined) {
            throw new Error(
              `attempted to access unknown env value: ${prop as any}`
            )
          }
          return val
        },
      }),
      contracts: new Proxy(contracts, {
        get: (target, prop) => {
          const val = target[prop]
          if (val === undefined) {
            throw new Error(
              `attempted to access unknown contract: ${prop as any}`
            )
          }
          return val
        },
      }),
    })
  )
}

/**
 * Generates a ChugSplash action bundle from a config file.
 * @param config Config file to convert into a bundle.
 * @param env Environment variables to inject into the config file.
 * @returns Action bundle generated from the parsed config file.
 */
export const makeActionBundleFromConfig = async (
  config: ChugSplashConfig,
  env: {
    [key: string]: string | number | boolean
  } = {}
): Promise<ChugSplashActionBundle> => {
  // Parse the config to replace any template variables.
  const parsed = parseChugSplashConfig(config, env)

  const actions: ChugSplashAction[] = []
  for (const [contractName, contractConfig] of Object.entries(
    parsed.contracts
  )) {
    const artifact = await getContractArtifact(contractConfig.source)
    const storageLayout = await getStorageLayout(contractConfig.source)

    // Add a SET_CODE action for each contract first.
    actions.push({
      target: contractConfig.address,
      code: artifact.deployedBytecode,
    })

    // Add SET_STORAGE actions for each storage slot that we want to modify.
    const slots = computeStorageSlots(storageLayout, contractConfig.variables)
    for (const slot of slots) {
      actions.push({
        target: contractConfig.address,
        key: slot.key,
        value: slot.val,
      })
    }
  }

  // Generate a bundle from the list of actions.
  return getChugSplashActionBundle(actions)
}
