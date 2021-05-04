import { Signer, Contract } from 'ethers'
import { Provider } from '@ethersproject/abstract-provider'
import { getL1ContractData, getL2ContractData } from './contract-data'

type Network = 'goerli' | 'kovan' | 'mainnet'

const checkSignerType = (signerOrProvider: Signer | Provider) => {
  if (!signerOrProvider) throw Error('signerOrProvider argument is undefined')
  if (
    !(signerOrProvider instanceof Signer) &&
    !(signerOrProvider instanceof Provider)
  )
    throw Error('signerOrProvider argument is the wrong type')
}

export const connectL1Contracts = async (
  signerOrProvider,
  network: Network
): Promise<{ [key: string]: Contract }> => {
  checkSignerType(signerOrProvider)

  if (!network) throw Error('network argument is undefined')
  if (network !== 'mainnet' && network !== 'kovan' && network !== 'goerli')
    throw Error('network argument is the wrong type')

  const l1ContractData = getL1ContractData(network)

  const namesMap = {
    Lib_AddressManager: 'addressManager',
    OVM_CanonicalTransactionChain: 'canonicalTransactionChain',
    OVM_ExecutionManager: 'executionManager',
    OVM_FraudVerifier: 'fraudVerifier',
    OVM_L1CrossDomainMessenger: 'xDomainMessenger',
    OVM_L1ETHGateway: 'ethGateway',
    OVM_L1MultiMessageRelayer: 'multiMessageRelayer',
    OVM_SafetyChecker: 'safetyChecker',
    OVM_StateCommitmentChain: 'stateCommitmentChain',
    OVM_StateManagerFactory: 'stateManagerFactory',
    OVM_StateTransitionerFactory: 'stateTransitionerFactory',
    Proxy__OVM_L1CrossDomainMessenger: 'xDomainMessengerProxy',
    Proxy__OVM_L1ETHGateway: 'l1EthGatewayProxy',
    mockOVM_BondManager: 'mockBondManger',
  }

  return Object.entries(l1ContractData).reduce(
    (allContracts, [contractName, contractData]) => {
      allContracts[namesMap[contractName]] = new Contract(
        contractData.address,
        contractData.abi,
        signerOrProvider
      )
      return allContracts
    },
    {}
  )
}

export const connectL2Contracts = async (
  signerOrProvider
): Promise<{ [key: string]: Contract }> => {
  const l2ContractData = await getL2ContractData()
  checkSignerType(signerOrProvider)

  const namesMap = {
    OVM_ETH: 'eth',
    OVM_L2CrossDomainMessenger: 'xDomainMessenger',
    OVM_L2ToL1MessagePasser: 'messagePasser',
    OVM_L1MessageSender: 'messageSender',
    OVM_DeployerWhitelist: 'deployerWhiteList',
    OVM_ECDSAContractAccount: 'ecdsaContractAccount',
    OVM_SequencerEntrypoint: 'sequencerEntrypoint',
    ERC1820Registry: 'erc1820Registry',
    Lib_AddressManager: 'addressManager',
  }

  return Object.entries(l2ContractData).reduce(
    (allContracts, [contractName, contractData]) => {
      allContracts[namesMap[contractName]] = new Contract(
        contractData.address,
        contractData.abi,
        signerOrProvider
      )
      return allContracts
    },
    {}
  )
}
