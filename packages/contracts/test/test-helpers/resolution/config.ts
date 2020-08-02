/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract } from 'ethers'
import { DEFAULT_GAS_METER_PARAMS } from '@eth-optimism/rollup-core'

/* Internal Imports */
import { AddressResolverDeployConfig, AddressResolverConfig } from './types'
import { GAS_LIMIT, DEFAULT_FORCE_INCLUSION_PERIOD } from '../constants'

/**
 * Generates the default deployment configuration. Runs as an async function
 * because we need to get the contract factories async via buidler.
 * @param addressResolver Address resolver contract to connect to.
 * @returns Default address resolver deployment configuration.
 */
export const getDefaultDeployConfig = async (
  addressResolver: Contract
): Promise<AddressResolverDeployConfig> => {
  const [owner, sequencer, l1ToL2TransactionPasser] = await ethers.getSigners()

  return {
    L1ToL2TransactionQueue: {
      factory: await ethers.getContractFactory('L1ToL2TransactionQueue'),
      params: [
        addressResolver.address,
        await l1ToL2TransactionPasser.getAddress(),
      ],
    },
    SafetyTransactionQueue: {
      factory: await ethers.getContractFactory('SafetyTransactionQueue'),
      params: [addressResolver.address],
    },
    CanonicalTransactionChain: {
      factory: await ethers.getContractFactory('CanonicalTransactionChain'),
      params: [
        addressResolver.address,
        await sequencer.getAddress(),
        await l1ToL2TransactionPasser.getAddress(),
        DEFAULT_FORCE_INCLUSION_PERIOD,
      ],
    },
    StateCommitmentChain: {
      factory: await ethers.getContractFactory('StateCommitmentChain'),
      params: [addressResolver.address],
    },
    StateManager: {
      factory: await ethers.getContractFactory('FullStateManager'),
      params: [],
    },
    ExecutionManager: {
      factory: await ethers.getContractFactory('ExecutionManager'),
      params: [addressResolver.address, await owner.getAddress(), DEFAULT_GAS_METER_PARAMS],
    },
    SafetyChecker: {
      factory: await ethers.getContractFactory('StubSafetyChecker'),
      params: [],
    },
    FraudVerifier: {
      factory: await ethers.getContractFactory('FraudVerifier'),
      params: [addressResolver.address],
    },
  }
}

/**
 * Generates the deployment configuration for various libraries.
 * @returns Library deployment configuration.
 */
export const getLibraryDeployConfig = async (): Promise<any> => {
  return {
    RollupMerkleUtils: {
      factory: await ethers.getContractFactory('RollupMerkleUtils'),
      params: [],
    },
  }
}

/**
 * Given a config, generates the default config and merges the two.
 * @param addressResolver Address resolver to connect to the config.
 * @param config User-provided configuration.
 * @returns Config merged with default config.
 */
export const makeDeployConfig = async (
  addressResolver: Contract,
  config: Partial<AddressResolverConfig>
): Promise<AddressResolverDeployConfig> => {
  const defaultDeployConfig = await getDefaultDeployConfig(addressResolver)

  return {
    ...defaultDeployConfig,
    ...config.deployConfig,
  }
}
