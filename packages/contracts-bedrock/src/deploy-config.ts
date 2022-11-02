import { ethers } from 'ethers'

/**
 * Helper type to make it easier to define a deploy config.
 */
type address = string

/**
 * Defines the configuration for a deployment.
 */
export interface DeployConfig {
  /**
   * Address of the account that will own the entire system after the deployment is completed.
   */
  systemOwner: address

  /**
   * Address of the account that will be the controller of the system dictator process.
   */
  controller: address

  /**
   * Tag, block number, or block hash of the L1 block that the L2 system will start at.
   */
  l1StartingBlockTag: string

  /**
   * Seconds that users need to wait until L2 state outputs are finalized.
   */
  finalizationPeriodSeconds: number

  /**
   * Seconds per L2 block.
   */
  l2BlockTimeSeconds: number

  /**
   * Number of L2 blocks between each submission of L2 state outputs.
   */
  l2OutputOracleSubmissionInterval: number

  /**
   * First L2 state output.
   */
  l2OutputOracleGenesisL2Output: string

  /**
   * Number of L2 blocks in the legacy system.
   */
  l2OutputOracleHistoricalTotalBlocks: number

  /**
   * First L2 block number.
   */
  l2OutputOracleStartingBlockNumber: number

  /**
   * Timestamp of the first L2 block.
   */
  l2OutputOracleStartingTimestamp: number

  /**
   * Address of the proposer of new L2 state outputs.
   */
  l2OutputOracleProposer: address

  /**
   * Address of the owner of the L2OutputOracle.
   */
  l2OutputOracleOwner: address

  /**
   * Number of L1 block confirmations to wait when deploying.
   */
  numDeployConfirmations: number
}

/**
 * Specification for each of the configuration options.
 */
export const deployConfigSpec: {
  [K in keyof DeployConfig]: {
    type: 'address' | 'number' | 'string'
    default?: any
  }
} = {
  systemOwner: {
    type: 'address',
    default: ethers.constants.AddressZero,
  },
  controller: {
    type: 'address',
    default: ethers.constants.AddressZero,
  },
  l1StartingBlockTag: {
    type: 'string',
  },
  finalizationPeriodSeconds: {
    type: 'number',
    default: 2,
  },
  l2BlockTimeSeconds: {
    type: 'number',
    default: 2,
  },
  l2OutputOracleSubmissionInterval: {
    type: 'number',
  },
  l2OutputOracleGenesisL2Output: {
    type: 'string',
    default: ethers.constants.HashZero,
  },
  l2OutputOracleHistoricalTotalBlocks: {
    type: 'number',
    default: 0,
  },
  l2OutputOracleStartingBlockNumber: {
    type: 'number',
    default: 0,
  },
  l2OutputOracleStartingTimestamp: {
    type: 'number',
  },
  l2OutputOracleProposer: {
    type: 'address',
  },
  l2OutputOracleOwner: {
    type: 'address',
  },
  numDeployConfirmations: {
    type: 'number',
    default: 1,
  },
}
