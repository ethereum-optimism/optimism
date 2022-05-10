import { ethers } from 'ethers'

/**
 * Defines the configuration for a deployment.
 */
export interface DeployConfig {
  /**
   * Whether or not this network is a forked network.
   */
  isForkedNetwork?: boolean

  /**
   * Optional number of confs to wait during deployment.
   */
  numDeployConfirmations?: number

  /**
   * Optional gas price to use for deployment transactions.
   */
  gasPrice?: number

  /**
   * Estimated average L1 block time in seconds.
   */
  l1BlockTimeSeconds: number

  /**
   * Gas limit for blocks on L2.
   */
  l2BlockGasLimit: number

  /**
   * Chain ID for the L2 network.
   */
  l2ChainId: number

  /**
   * Discount divisor used to calculate gas burn for L1 to L2 transactions.
   */
  ctcL2GasDiscountDivisor: number

  /**
   * Cost of the "enqueue" function in the CTC.
   */
  ctcEnqueueGasCost: number

  /**
   * Fault proof window in seconds.
   */
  sccFaultProofWindowSeconds: number

  /**
   * Sequencer publish window in seconds.
   */
  sccSequencerPublishWindowSeconds: number

  /**
   * Address of the Sequencer (publishes to CTC).
   */
  ovmSequencerAddress: string

  /**
   * Address of the Proposer (publishes to SCC).
   */
  ovmProposerAddress: string

  /**
   * Address of the account that will sign blocks.
   */
  ovmBlockSignerAddress: string

  /**
   * Address that will receive fees on L1.
   */
  ovmFeeWalletAddress: string

  /**
   * Address of the owner of the AddressManager contract on L1.
   */
  ovmAddressManagerOwner: string

  /**
   * Address of the owner of the GasPriceOracle contract on L2.
   */
  ovmGasPriceOracleOwner: string

  /**
   * Optional whitelist owner address.
   */
  ovmWhitelistOwner?: string

  /**
   * Optional initial overhead value for GPO (default: 2750).
   */
  gasPriceOracleOverhead?: number

  /**
   * Optional initial scalar value for GPO (default: 1500000).
   */
  gasPriceOracleScalar?: number

  /**
   * Optional initial decimals for GPO (default: 6).
   */
  gasPriceOracleDecimals?: number

  /**
   * Optional initial L1 base fee for GPO (default: 1).
   */
  gasPriceOracleL1BaseFee?: number

  /**
   * Optional initial L2 gas price for GPO (default: 1).
   */
  gasPriceOracleL2GasPrice?: number

  /**
   * Optional block number to enable the Berlin hardfork (default: 0).
   */
  hfBerlinBlock?: number
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
  isForkedNetwork: {
    type: 'boolean',
    default: false,
  },
  numDeployConfirmations: {
    type: 'number',
    default: 0,
  },
  gasPrice: {
    type: 'number',
    default: undefined,
  },
  l1BlockTimeSeconds: {
    type: 'number',
  },
  l2BlockGasLimit: {
    type: 'number',
  },
  l2ChainId: {
    type: 'number',
  },
  ctcL2GasDiscountDivisor: {
    type: 'number',
  },
  ctcEnqueueGasCost: {
    type: 'number',
  },
  sccFaultProofWindowSeconds: {
    type: 'number',
  },
  sccSequencerPublishWindowSeconds: {
    type: 'number',
  },
  ovmSequencerAddress: {
    type: 'address',
  },
  ovmProposerAddress: {
    type: 'address',
  },
  ovmBlockSignerAddress: {
    type: 'address',
  },
  ovmFeeWalletAddress: {
    type: 'address',
  },
  ovmAddressManagerOwner: {
    type: 'address',
  },
  ovmGasPriceOracleOwner: {
    type: 'address',
  },
  ovmWhitelistOwner: {
    type: 'address',
    default: ethers.constants.AddressZero,
  },
  gasPriceOracleOverhead: {
    type: 'number',
    default: 2750,
  },
  gasPriceOracleScalar: {
    type: 'number',
    default: 1_500_000,
  },
  gasPriceOracleDecimals: {
    type: 'number',
    default: 6,
  },
  gasPriceOracleL1BaseFee: {
    type: 'number',
    default: 1,
  },
  gasPriceOracleL2GasPrice: {
    type: 'number',
    default: 1,
  },
  hfBerlinBlock: {
    type: 'number',
    default: 0,
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
    config = require(`../deploy-config/${network}.ts`).default
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
