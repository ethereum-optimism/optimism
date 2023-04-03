import * as path from 'path'
import * as fs from 'fs'

import { extendEnvironment, extendConfig } from 'hardhat/config'
import {
  HardhatConfig,
  HardhatRuntimeEnvironment,
  HardhatUserConfig,
} from 'hardhat/types'
import { lazyObject, lazyFunction } from 'hardhat/plugins'
import { ethers } from 'ethers'

// From: https://github.com/wighawag/hardhat-deploy/blob/master/src/index.ts#L63-L76
const normalizePath = (
  config: HardhatConfig,
  userPath: string | undefined,
  defaultPath: string
): string => {
  if (userPath === undefined) {
    userPath = path.join(config.paths.root, defaultPath)
  } else {
    if (!path.isAbsolute(userPath)) {
      userPath = path.normalize(path.join(config.paths.root, userPath))
    }
  }
  return userPath
}

const getDeployConfig = (
  dir: string,
  network: string
): { [key: string]: any } => {
  let config: any
  try {
    const base = `${dir}/${network}`
    if (fs.existsSync(`${base}.ts`)) {
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      config = require(`${base}.ts`).default
    } else if (fs.existsSync(`${base}.json`)) {
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      config = require(`${base}.json`)
    } else {
      throw new Error('not found')
    }
  } catch (err) {
    throw new Error(
      `error while loading deploy config for network: ${network}, ${err}`
    )
  }
  return config
}

export const loadDeployConfig = (hre: HardhatRuntimeEnvironment): any => {
  const paths = hre.config.paths.deployConfig
  const conf = getDeployConfig(paths, hre.network.name)
  const spec = parseDeployConfig(hre, conf)

  return new Proxy(spec, {
    get: (target, prop) => {
      if (target.hasOwnProperty(prop)) {
        return target[prop]
      }

      // Explicitly throw if the property is not found
      throw new Error(
        `property does not exist in deploy config: ${String(prop)}`
      )
    },
  })
}

export const parseDeployConfig = (
  hre: HardhatRuntimeEnvironment,
  config: any
): any => {
  // Create a clone of the config object. Shallow clone is fine because none of the input options
  // are expected to be objects or functions etc.
  const parsed = { ...config }

  // If the deployConfigSpec is not provided, do no validation
  if (!hre.config.deployConfigSpec) {
    return parsed
  }

  for (const [key, spec] of Object.entries(hre.config.deployConfigSpec)) {
    // Make sure the value is defined, or use a default.
    if (parsed[key] === undefined) {
      if ('default' in spec) {
        parsed[key] = spec.default
      } else {
        throw new Error(
          `deploy config is missing required field: ${key} (${spec.type})`
        )
      }
    } else {
      // Make sure the default has the correct type.
      if (spec.type === 'address') {
        if (!ethers.utils.isAddress(parsed[key])) {
          throw new Error(
            `deploy config field: ${key} is not of type ${spec.type}: ${parsed[key]}`
          )
        }
      } else if (typeof parsed[key] !== spec.type) {
        throw new Error(
          `deploy config field: ${key} is not of type ${spec.type}: ${parsed[key]}`
        )
      }
    }
  }

  return parsed
}

extendConfig(
  (config: HardhatConfig, userConfig: Readonly<HardhatUserConfig>) => {
    config.paths.deployConfig = normalizePath(
      config,
      userConfig.paths?.deployConfig,
      'deploy-config'
    )
  }
)

extendEnvironment((hre) => {
  hre.deployConfig = lazyObject(() => loadDeployConfig(hre))
  hre.getDeployConfig = lazyFunction(() => {
    const paths = hre.config.paths.deployConfig
    return (network: string) => getDeployConfig(paths, network)
  })
})
