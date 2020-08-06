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
  | 'L1ToL2TransactionQueue'
  | 'SafetyTransactionQueue'
  | 'CanonicalTransactionChain'
  | 'StateCommitmentChain'
  | 'StateManager'
  | 'ExecutionManager'
  | 'SafetyChecker'
  | 'FraudVerifier'
  | 'RollupMerkleUtils'

export interface ContractDeployConfig {
  L1ToL2TransactionQueue: ContractDeployOptions
  SafetyTransactionQueue: ContractDeployOptions
  CanonicalTransactionChain: ContractDeployOptions
  StateCommitmentChain: ContractDeployOptions
  StateManager: ContractDeployOptions
  ExecutionManager: ContractDeployOptions
  SafetyChecker: ContractDeployOptions
  FraudVerifier: ContractDeployOptions
  RollupMerkleUtils: ContractDeployOptions
}

interface ContractMapping {
  l1ToL2TransactionQueue: Contract
  safetyTransactionQueue: Contract
  canonicalTransactionChain: Contract
  stateCommitmentChain: Contract
  stateManager: Contract
  executionManager: Contract
  safetyChecker: Contract
  fraudVerifier: Contract
  rollupMerkleUtils: Contract
}

export interface AddressResolverMapping {
  addressResolver: Contract
  contracts: ContractMapping
}

export const factoryToContractName = {
  L1ToL2TransactionQueue: 'l1ToL2TransactionQueue',
  SafetyTransactionQueue: 'safetyTransactionQueue',
  CanonicalTransactionChain: 'canonicalTransactionChain',
  StateCommitmentChain: 'stateCommitmentChain',
  StateManager: 'stateManager',
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
