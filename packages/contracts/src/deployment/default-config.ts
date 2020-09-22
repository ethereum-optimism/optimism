/* External Imports */
import { Contract, Signer } from 'ethers'

/* Internal Imports */
import { getContractFactory } from '../contract-imports'
import { ContractDeployConfig, RollupOptions } from './types'

/**
 * Generates the default deployment configuration. Runs as an async function
 * because we need to get the contract factories async via buidler.
 * @param addressResolverAddress The address of the AddressResolver contract.
 * @param deployerWallet The Signer used to deploy contracts.
 * @param rollupOptions The RollupOptions to use to configure the contracts to deploy.
 * @returns Default address resolver deployment configuration.
 */
export const getDefaultContractDeployConfig = async (
  addressResolverAddress: string,
  deployerWallet: Signer,
  rollupOptions: RollupOptions
): Promise<ContractDeployConfig> => {
  return {
    GasConsumer: {
      factory: await getContractFactory('GasConsumer'),
      params: [],
      signer: deployerWallet,
    },
    DeployerWhitelist: {
      factory: await getContractFactory('DeployerWhitelist'),
      params: [
        rollupOptions.deployerWhitelistOwnerAddress,
        rollupOptions.allowArbitraryContractDeployment,
      ],
      signer: deployerWallet,
    },
    L1ToL2TransactionQueue: {
      factory: getContractFactory('L1ToL2TransactionQueue'),
      params: [addressResolverAddress],
      signer: deployerWallet,
    },
    SafetyTransactionQueue: {
      factory: getContractFactory('SafetyTransactionQueue'),
      params: [addressResolverAddress],
      signer: deployerWallet,
    },
    CanonicalTransactionChain: {
      factory: getContractFactory('CanonicalTransactionChain'),
      params: [
        addressResolverAddress,
        rollupOptions.sequencerAddress,
        rollupOptions.forceInclusionPeriodSeconds,
      ],
      signer: deployerWallet,
    },
    StateCommitmentChain: {
      factory: getContractFactory('StateCommitmentChain'),
      params: [addressResolverAddress],
      signer: deployerWallet,
    },
    StateManager: {
      factory: getContractFactory('FullStateManager'),
      params: [],
      signer: deployerWallet,
    },
    ExecutionManager: {
      factory: getContractFactory('ExecutionManager'),
      params: [
        addressResolverAddress,
        rollupOptions.ownerAddress,
        [
          rollupOptions.gasMeterConfig.ovmTxFlatGasFee,
          rollupOptions.gasMeterConfig.ovmTxMaxGas,
          rollupOptions.gasMeterConfig.gasRateLimitEpochLength,
          rollupOptions.gasMeterConfig.maxSequencedGasPerEpoch,
          rollupOptions.gasMeterConfig.maxQueuedGasPerEpoch,
        ],
      ],
      signer: deployerWallet,
    },
    SafetyChecker: {
      factory: getContractFactory('StubSafetyChecker'),
      params: [],
      signer: deployerWallet,
    },
    FraudVerifier: {
      factory: getContractFactory('FraudVerifier'),
      params: [addressResolverAddress],
      signer: deployerWallet,
    },
    RollupMerkleUtils: {
      factory: getContractFactory('RollupMerkleUtils'),
      params: [],
      signer: deployerWallet,
    },
  }
}

/**
 * Merges the given config with the default config.
 * @param addressResolverAddress The address of the AddressResolver contract.
 * @param config Config to merge with default.
 * @param signer Signer to use to deploy contracts.
 * @param options Rollup chain options.
 */
export const mergeDefaultConfig = async (
  addressResolverAddress: string,
  config?: Partial<ContractDeployConfig>,
  signer?: Signer,
  options?: RollupOptions
): Promise<ContractDeployConfig> => {
  const defaultConfig = await getDefaultContractDeployConfig(
    addressResolverAddress,
    signer,
    options
  )

  return {
    ...defaultConfig,
    ...(config || {}),
  }
}
