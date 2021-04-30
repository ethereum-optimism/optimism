import Lib_L1AddressManager from './mainnet-v2/Lib_AddressManager.json'
import OVM_CanonicalTransactionChain from './mainnet-v2/OVM_CanonicalTransactionChain.json'
import OVM_ExecutionManager from './mainnet-v2/OVM_ExecutionManager.json'
import OVM_FraudVerifier from './mainnet-v2/OVM_FraudVerifier.json'
import OVM_L1CrossDomainMessenger from './mainnet-v2/OVM_L1CrossDomainMessenger.json'
import OVM_L1ETHGateway from './mainnet-v2/OVM_L1ETHGateway.json'
import OVM_L1MultiMessageRelayer from './mainnet-v2/OVM_L1MultiMessageRelayer.json'
import OVM_SafetyChecker from './mainnet-v2/OVM_SafetyChecker.json'
import OVM_StateCommitmentChain from './mainnet-v2/OVM_StateCommitmentChain.json'
import OVM_StateManagerFactory from './mainnet-v2/OVM_StateManagerFactory.json'
import OVM_StateTransitionerFactory from './mainnet-v2/OVM_StateTransitionerFactory.json'
import Proxy__OVM_L1CrossDomainMessenger from './mainnet-v2/Proxy__OVM_L1CrossDomainMessenger.json'
import Proxy__OVM_L1ETHGateway from './mainnet-v2/Proxy__OVM_L1ETHGateway.json'
import mockOVM_BondManager from './mainnet-v2/mockOVM_BondManager.json'

export interface Layer1ContractsType {
  addressManager: typeof Lib_L1AddressManager.abi
  canonicalTransactionChain: typeof OVM_CanonicalTransactionChain.abi
  executionManager: typeof OVM_ExecutionManager.abi
  fraudVerifier: typeof OVM_FraudVerifier.abi
  xDomainMessenger: typeof OVM_L1CrossDomainMessenger.abi
  xDomainMessengerProxy: typeof Proxy__OVM_L1CrossDomainMessenger.abi
  ethGateway: typeof OVM_L1ETHGateway.abi
  ethGatewayProxy: typeof Proxy__OVM_L1ETHGateway.abi
  multiMessageRelayer: typeof OVM_L1MultiMessageRelayer.abi
  safetyChecker: typeof OVM_SafetyChecker.abi
  stateCommitmentChain: typeof OVM_StateCommitmentChain.abi
  stateManagerFactory: typeof OVM_StateManagerFactory.abi
  stateTransitionerFactory: typeof OVM_StateTransitionerFactory.abi
  mockBondManager: typeof mockOVM_BondManager.abi
}

export const getL1ContractData = (network: 'goerli' | 'kovan' | 'mainnet') => {
  return {
    Lib_L1AddressManager: require(`../deployments/${network}-v2/Lib_AddressManager.json`),
    OVM_CanonicalTransactionChain: require(`../deployments/${network}-v2/OVM_CanonicalTransactionChain.json`),
    OVM_ExecutionManager: require(`../deployments/${network}-v2/OVM_ExecutionManager.json`),
    OVM_FraudVerifier: require(`../deployments/${network}-v2/OVM_FraudVerifier.json`),
    OVM_L1CrossDomainMessenger: require(`../deployments/${network}-v2/OVM_L1CrossDomainMessenger.json`),
    OVM_L1ETHGateway: require(`../deployments/${network}-v2/OVM_L1ETHGateway.json`),
    OVM_L1MultiMessageRelayer: require(`../deployments/${network}-v2/OVM_L1MultiMessageRelayer.json`),
    OVM_SafetyChecker: require(`../deployments/${network}-v2/OVM_SafetyChecker.json`),
    OVM_StateCommitmentChain: require(`../deployments/${network}-v2/OVM_StateCommitmentChain.json`),
    OVM_StateManagerFactory: require(`../deployments/${network}-v2/OVM_StateManagerFactory.json`),
    OVM_StateTransitionerFactory: require(`../deployments/${network}-v2/OVM_StateTransitionerFactory.json`),
    Proxy__OVM_L1CrossDomainMessenger: require(`../deployments/${network}-v2/Proxy__OVM_L1CrossDomainMessenger.json`),
    Proxy__OVM_L1ETHGateway: require(`../deployments/${network}-v2/Proxy__OVM_L1ETHGateway.json`),
    mockOVM_BondManager: require(`../deployments/${network}-v2/mockOVM_BondManager.json`),
  }
}
