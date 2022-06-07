import { ethers } from 'ethers'

/**
 * Defines the configuration for a deployment.
 */
export interface DeployConfig {
  /**
   * Initial RetroReceiver owner.
   */
  retroReceiverOwner: string

  /**
   * Initial Drippie owner.
   */
  drippieOwner: string
}

/**
 * Specification for each of the configuration options.
 */
const configSpec: {
  [K in keyof DeployConfig]: {
    type: string
    default?: any
  }
} = {
  retroReceiverOwner: {
    type: 'address',
  },
  drippieOwner: {
    type: 'address',
  },
}

/**
 * Gets the deploy config for the given network.
 *
 * @param network Network name.
 * @returns Deploy config for the given network.
 */
export const getDeployConfig = (network: string): Required<DeployConfig> => {
  let config: DeployConfig
  try {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    config = require(`../../config/deploy/${network}.ts`).default
  } catch (err) {
    throw new Error(
      `error while loading deploy config for network: ${network}, ${err}`
    )
  }

  return parseDeployConfig(config)
}

/**
 * Parses and validates the given deploy config, replacing any missing values with defaults.
 *
 * @param config Deploy config to parse.
 * @returns Parsed deploy config.
 */
export const parseDeployConfig = (
  config: DeployConfig
): Required<DeployConfig> => {
  // Create a clone of the config object. Shallow clone is fine because none of the input options
  // are expected to be objects or functions etc.
  const parsed = { ...config }

  for (const [key, spec] of Object.entries(configSpec)) {
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

  return parsed as Required<DeployConfig>
}
