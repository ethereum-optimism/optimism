/* External Imports */
import { Contract, ContractFactory, Signer } from 'ethers'

export interface ContractDeployOptions {
  factory: ContractFactory
  params: any[]
  signer: Signer
}

export interface RollupOptions {
  gasLimit: number
  forceInclusionPeriod: number
  owner: Signer
  sequencer: Signer
  l1ToL2TransactionPasser: Signer
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
  | 'ContractAddressGenerator'
  | 'EthMerkleTrie'
  | 'RLPEncode'
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
  ContractAddressGenerator: ContractDeployOptions
  EthMerkleTrie: ContractDeployOptions
  RLPEncode: ContractDeployOptions
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
  contractAddressGenerator: Contract
  ethMerkleTrie: Contract
  rlpEncode: Contract
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
  ContractAddressGenerator: 'contractAddressGenerator',
  EthMerkleTrie: 'ethMerkleTrie',
  RLPEncode: 'rlpEncode',
  RollupMerkleUtils: 'rollupMerkleUtils'
}

export interface RollupDeployConfig {
  signer: Signer
  rollupOptions: RollupOptions
  addressResolverConfig?: ContractDeployOptions
  contractDeployConfig?: ContractDeployConfig
  dependencies?: ContractFactoryName[]
}