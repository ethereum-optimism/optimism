/* External Imports */
import hre from 'hardhat'
import { cloneDeep, isPlainObject } from 'lodash'
import { ethers } from 'ethers'

type SolidityVariable =
  | string
  | number
  | Array<SolidityVariable>
  | {
      [name: string]: SolidityVariable
    }

export interface ChugSplashConfig {
  contracts: {
    [name: string]: {
      source: string
      variables?: {
        [name: string]: SolidityVariable
      }
    }
  }
}

/**
 * Parses any template strings found inside of a variable. Will parse recursively if the variable
 * is an array or a plain object. Keys inside plain objects can also be templated in.
 * @param variable Variable to replace.
 * @param env Environment variables to inject into {{ env.X }} template strings.
 * @param addresses Contract addresses to inject into {{ contract.X }} template strings.
 * @returns Modified variable with template strings replaced.
 */
const parseVariable = (
  variable: SolidityVariable,
  env: {
    [name: string]: string
  } = {},
  addresses: {
    [name: string]: string
  } = {}
): SolidityVariable => {
  if (typeof variable === 'string') {
    // "{{ }}" is a template string and needs to be replaced with the desired value.
    const match = /{{ (.*?) }}/gm.exec(variable)
    if (match && match.length == 2) {
      if (match[1].startsWith('env.')) {
        const templateKey = match[1].replace('env.', '')
        const templateVal = env[templateKey]
        if (templateVal === undefined) {
          throw new Error(
            `[chugsplash]: key does not exist in environment: ${templateKey}`
          )
        } else {
          return templateVal
        }
      } else if (match[1].startsWith('contracts.')) {
        const templateKey = match[1].replace('contracts.', '')
        const templateVal = addresses[templateKey]
        if (templateVal === undefined) {
          throw new Error(
            `[chugsplash]: contract does not exist: ${templateKey}`
          )
        } else {
          return templateVal
        }
      } else {
        throw new Error(
          `[chugsplash]: unrecognized template string: ${variable}`
        )
      }
    } else {
      return variable
    }
  } else if (Array.isArray(variable)) {
    // Each array element gets parsed individually.
    return variable.map((element) => {
      return parseVariable(element, env)
    })
  } else if (isPlainObject(variable)) {
    // Parse the keys *and* values for objects.
    variable = cloneDeep(variable)
    for (const [key, val] of Object.entries(variable)) {
      delete variable[key] // Make sure to delete the original key!
      variable[parseVariable(key, env) as string] = parseVariable(val, env)
    }
    return variable
  } else {
    // Anything else just gets returned as-is.
    return variable
  }
}

// TODO: Change this when we break this logic out into its own package.
const proxyArtifact = hre.artifacts.readArtifactSync('ChugSplashProxy')

/**
 * Replaces any template strings inside of a chugsplash config.
 * @param config Config to update with template strings.
 * @param env Environment variables to inject into {{ env.X }}.
 * @returns Config with any template strings replaced.
 */
export const parseConfig = (
  config: ChugSplashConfig,
  deployerAddress: string,
  env: any = {}
): ChugSplashConfig => {
  // TODO: Might want to do config validation here.

  // Make a copy of the config so that we can modify it without accidentally modifying the
  // original object.
  const parsed = cloneDeep(config)

  // Generate a mapping of contract names to contract addresses. Used to inject values for
  // {{ contract.X }} template strings.
  const addresses = {}
  for (const contractNickname of Object.keys(config.contracts || {})) {
    addresses[contractNickname] = ethers.utils.getCreate2Address(
      deployerAddress,
      ethers.utils.keccak256(ethers.utils.toUtf8Bytes(contractNickname)),
      ethers.utils.keccak256(proxyArtifact.bytecode)
    )
  }

  for (const [contractNickname, contractConfig] of Object.entries(
    config.contracts || {}
  )) {
    for (const [variableName, variableValue] of Object.entries(
      contractConfig.variables || {}
    )) {
      parsed.contracts[contractNickname].variables[
        variableName
      ] = parseVariable(variableValue, env, addresses)
    }
  }

  return parsed
}
