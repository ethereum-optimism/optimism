/* External Imports */
import { Contract, Signer } from 'ethers'

/* Internal Imports */
import { getContractFactory } from '../contract-imports'
import { ContractDeployConfig, RollupOptions } from './types'

/**
 * Generates the default deployment configuration. Runs as an async function
 * because we need to get the contract factories async via buidler.
 * @param addressResolver Address resolver contract to connect to.
 * @returns Default address resolver deployment configuration.
 */
export const getDefaultContractDeployConfig = async (
  addressResolver: Contract,
  wallet: Signer,
  options: RollupOptions
): Promise<ContractDeployConfig> => {
  return {
    L1ToL2TransactionQueue: {
      factory: getContractFactory('L1ToL2TransactionQueue'),
      params: [
        addressResolver.address,
        await options.l1ToL2TransactionPasser.getAddress(),
      ],
      signer: wallet,
    },
    SafetyTransactionQueue: {
      factory: getContractFactory('SafetyTransactionQueue'),
      params: [addressResolver.address],
      signer: wallet,
    },
    CanonicalTransactionChain: {
      factory: getContractFactory('CanonicalTransactionChain'),
      params: [
        addressResolver.address,
        await options.sequencer.getAddress(),
        await options.l1ToL2TransactionPasser.getAddress(),
        options.forceInclusionPeriod,
      ],
      signer: wallet,
    },
    StateCommitmentChain: {
      factory: getContractFactory('StateCommitmentChain'),
      params: [addressResolver.address],
      signer: wallet,
    },
    StateManager: {
      factory: getContractFactory('FullStateManager'),
      params: [],
      signer: wallet,
    },
    ExecutionManager: {
      factory: getContractFactory('ExecutionManager'),
      params: [
        addressResolver.address,
        await options.owner.getAddress(),
        [
          options.gasMeterConfig.ovmTxFlatGasFee,
          options.gasMeterConfig.ovmTxMaxGas,
          options.gasMeterConfig.gasRateLimitEpochLength,
          options.gasMeterConfig.maxSequencedGasPerEpoch,
          options.gasMeterConfig.maxQueuedGasPerEpoch
        ]
      ],
      signer: wallet,
    },
    SafetyChecker: {
      factory: getContractFactory('StubSafetyChecker'),
      params: [],
      signer: wallet,
    },
    FraudVerifier: {
      factory: getContractFactory('FraudVerifier'),
      params: [addressResolver.address, true],
      signer: wallet,
    },
    ContractAddressGenerator: {
      factory: getContractFactory('ContractAddressGenerator'),
      params: [],
      signer: wallet,
    },
    EthMerkleTrie: {
      factory: getContractFactory('EthMerkleTrie'),
      params: [],
      signer: wallet,
    },
    RLPEncode: {
      factory: getContractFactory('RLPEncode'),
      params: [],
      signer: wallet,
    },
    RollupMerkleUtils: {
      factory: getContractFactory('RollupMerkleUtils'),
      params: [],
      signer: wallet,
    },
  }
}

/**
 * Merges the given config with the default config.
 * @param config Config to merge with default.
 * @param addressResolver AddressResolver contract reference.
 * @param signer Signer to use to deploy contracts.
 * @param options Rollup chain options.
 */
export const mergeDefaultConfig = async (
  config: Partial<ContractDeployConfig>,
  addressResolver: Contract,
  signer?: Signer,
  options?: RollupOptions
): Promise<ContractDeployConfig> => {
  const defaultConfig = await getDefaultContractDeployConfig(
    addressResolver,
    signer,
    options
  )
  return {
    ...defaultConfig,
    ...config,
  }
}
