import * as path from 'path'
import { predeploys as l2Addresses } from './predeploys'
import { Network } from './connect-contracts'

export const getL1ContractData = (network: Network) => {
  const contractNames = [
    'Lib_AddressManager',
    'OVM_CanonicalTransactionChain',
    'OVM_ExecutionManager',
    'OVM_FraudVerifier',
    'OVM_L1CrossDomainMessenger',
    'OVM_L1ETHGateway',
    'OVM_L1MultiMessageRelayer',
    'OVM_SafetyChecker',
    'OVM_StateCommitmentChain',
    'OVM_StateManagerFactory',
    'OVM_StateTransitionerFactory',
    'Proxy__OVM_L1CrossDomainMessenger',
    'Proxy__OVM_L1ETHGateway',
    'mockOVM_BondManager',
  ]
  return contractNames.reduce(
    (
      contractData: { [key: string]: { address: string; abi: any[] } },
      contractName
    ) => {
      contractData[contractName] = require(path.resolve(
        __dirname,
        `../deployments/${network}/${contractName}.json`
      ))
      return contractData
    },
    {}
  )
}

export const getL2ContractData = () => {
  return {
    OVM_ETH: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_ETH.sol/OVM_ETH.json`
      )).abi,
      address: l2Addresses.OVM_ETH,
    },
    OVM_L2CrossDomainMessenger: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/bridge/messaging/OVM_L2CrossDomainMessenger.sol/OVM_L2CrossDomainMessenger.json`
      )).abi,
      address: l2Addresses.OVM_L2CrossDomainMessenger,
    },
    OVM_L2ToL1MessagePasser: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_L2ToL1MessagePasser.sol/OVM_L2ToL1MessagePasser.json`
      )).abi,
      address: l2Addresses.OVM_L2ToL1MessagePasser,
    },
    OVM_L1MessageSender: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_L1MessageSender.sol/OVM_L1MessageSender.json`
      )).abi,
      address: l2Addresses.OVM_L1MessageSender,
    },
    OVM_DeployerWhitelist: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_DeployerWhitelist.sol/OVM_DeployerWhitelist.json`
      )).abi,
      address: l2Addresses.OVM_DeployerWhitelist,
    },
    OVM_ECDSAContractAccount: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/accounts/OVM_ECDSAContractAccount.sol/OVM_ECDSAContractAccount.json`
      )).abi,
      address: l2Addresses.OVM_ECDSAContractAccount,
    },
    OVM_SequencerEntrypoint: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_SequencerEntrypoint.sol/OVM_SequencerEntrypoint.json`
      )).abi,
      address: l2Addresses.OVM_SequencerEntrypoint,
    },
    ERC1820Registry: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/ERC1820Registry.sol/ERC1820Registry.json`
      )).abi,
      address: l2Addresses.ERC1820Registry,
    },
    Lib_AddressManager: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/libraries/resolver/Lib_AddressManager.sol/Lib_AddressManager.json`
      )).abi,
      address: l2Addresses.Lib_AddressManager,
    },
  }
}
