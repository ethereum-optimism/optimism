/* External Imports */
import { Contract, ContractFactory, Signer } from 'ethers'

export interface ContractDeployOptions {
  factory: ContractFactory
  params: any[]
  signer: Signer
}

export interface GasMeterOptions {
  ovmTxFlatGasFee: number
  ovmTxMaxGas: number
  gasRateLimitEpochLength: number
  maxSequencedGasPerEpoch: number
  maxQueuedGasPerEpoch: number
}

export interface RollupOptions {
  forceInclusionPeriodSeconds: number
  ownerAddress: string
  sequencerAddress: string
  gasMeterConfig: GasMeterOptions
}

export type ContractFactoryName =
  | 'GasConsumer'
  | 'L1ToL2TransactionQueue'
  | 'SafetyTransactionQueue'
  | 'CanonicalTransactionChain'
  | 'StateCommitmentChain'
  | 'StateManager'
  | 'StateManagerGasSanitizer'
  | 'ExecutionManager'
  | 'SafetyChecker'
  | 'FraudVerifier'
  | 'RollupMerkleUtils'

export interface ContractDeployConfig {
  GasConsumer: ContractDeployOptions
  L1ToL2TransactionQueue: ContractDeployOptions
  SafetyTransactionQueue: ContractDeployOptions
  CanonicalTransactionChain: ContractDeployOptions
  StateCommitmentChain: ContractDeployOptions
  StateManager: ContractDeployOptions
  StateManagerGasSanitizer: ContractDeployOptions
  ExecutionManager: ContractDeployOptions
  SafetyChecker: ContractDeployOptions
  FraudVerifier: ContractDeployOptions
  RollupMerkleUtils: ContractDeployOptions
}

interface ContractMapping {
  gasConsumer: Contract
  l1ToL2TransactionQueue: Contract
  safetyTransactionQueue: Contract
  canonicalTransactionChain: Contract
  stateCommitmentChain: Contract
  stateManager: Contract
  stateManagerGasSanitizer: Contract
  executionManager: Contract
  safetyChecker: Contract
  fraudVerifier: Contract
  rollupMerkleUtils: Contract
}

export interface AddressResolverMapping {
  addressResolver: Contract
  contracts: ContractMapping
}

export interface DeployResult extends AddressResolverMapping {
  failedDeployments: ContractFactoryName[]
}

export const factoryToContractName = {
  GasConsumer: 'gasConsumer',
  L1ToL2TransactionQueue: 'l1ToL2TransactionQueue',
  SafetyTransactionQueue: 'safetyTransactionQueue',
  CanonicalTransactionChain: 'canonicalTransactionChain',
  StateCommitmentChain: 'stateCommitmentChain',
  StateManager: 'stateManager',
  StateManagerGasSanitizer: 'stateManagerGasSanitizer',
  ExecutionManager: 'executionManager',
  SafetyChecker: 'safetyChecker',
  FraudVerifier: 'fraudVerifier',
  RollupMerkleUtils: 'rollupMerkleUtils',
}

export interface RollupDeployConfig {
  signer: Signer
  rollupOptions: RollupOptions
  addressResolverContractAddress?: string
  addressResolverConfig?: ContractDeployOptions
  contractDeployConfig?: Partial<ContractDeployConfig>
  dependencies?: ContractFactoryName[]
}
