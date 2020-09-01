/* External Imports */
import { Contract, ContractFactory } from 'ethers'

export interface ContractDeployConfig {
  factory: ContractFactory
  params: any[]
}

type ContractFactoryName =
  | 'GasConsumer'
  | 'L1ToL2TransactionQueue'
  | 'SafetyTransactionQueue'
  | 'CanonicalTransactionChain'
  | 'StateCommitmentChain'
  | 'StateManager'
  | 'ExecutionManager'
  | 'SafetyChecker'
  | 'FraudVerifier'
  | 'StateManagerGasSanitizer'
  | 'ECDSAContractAccount'

export interface AddressResolverDeployConfig {
  GasConsumer: ContractDeployConfig
  L1ToL2TransactionQueue: ContractDeployConfig
  SafetyTransactionQueue: ContractDeployConfig
  CanonicalTransactionChain: ContractDeployConfig
  StateCommitmentChain: ContractDeployConfig
  StateManager: ContractDeployConfig
  StateManagerGasSanitizer: ContractDeployConfig
  ExecutionManager: ContractDeployConfig
  SafetyChecker: ContractDeployConfig
  FraudVerifier: ContractDeployConfig
  ECDSAContractAccount: ContractDeployConfig
}

export interface AddressResolverConfig {
  deployConfig: AddressResolverDeployConfig
  dependencies: ContractFactoryName[]
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
  ecdsaContractAccount: Contract
}

export interface AddressResolverMapping {
  addressResolver: Contract
  contracts: ContractMapping
}

export const factoryToContractName = {
  GasConsumer: 'gasConsumer',
  L1ToL2TransactionQueue: 'l1ToL2TransactionQueue',
  SafetyTransactionQueue: 'safetyTransactionQueue',
  CanonicalTransactionChain: 'canonicalTransactionChain',
  StateCommitmentChain: 'stateCommitmentChain',
  StateManager: 'stateManager',
  StateManagerGasSanitizer: 'StateManagerGasSanitizer',
  ExecutionManager: 'executionManager',
  SafetyChecker: 'safetyChecker',
  FraudVerifier: 'fraudVerifier',
  ECDSAContractAccount: 'ecdsaContractAccount',
}
