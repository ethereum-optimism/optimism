import { Signer, Contract, providers } from 'ethers'
import { getL1ContractData, getL2ContractData } from './contract-data'

export const connectL1Contracts = async (
  signerOrProvider: Signer | providers.Provider,
  network: 'goerli' | 'kovan' | 'mainnet'
): Promise<Object> => {
  const l1ContractData = getL1ContractData(network)

  return {
    addressManager: new Contract(
      l1ContractData.Lib_AddressManager.abi,
      l1ContractData.Lib_AddressManager.address,
      signerOrProvider
    ),
    canonicalTransactionChain: new Contract(
      l1ContractData.OVM_CanonicalTransactionChain.abi,
      l1ContractData.OVM_CanonicalTransactionChain.address,
      signerOrProvider
    ),
    executionManager: new Contract(
      l1ContractData.OVM_ExecutionManager.abi,
      l1ContractData.OVM_ExecutionManager.address,
      signerOrProvider
    ),
    fraudVerifier: new Contract(
      l1ContractData.OVM_FraudVerifier.abi,
      l1ContractData.OVM_FraudVerifier.address,
      signerOrProvider
    ),
    xDomainMessenger: new Contract(
      l1ContractData.OVM_L1CrossDomainMessenger.abi,
      l1ContractData.OVM_L1CrossDomainMessenger.address,
      signerOrProvider
    ),
    ethGateway: new Contract(
      l1ContractData.OVM_L1ETHGateway.abi,
      l1ContractData.OVM_L1ETHGateway.address,
      signerOrProvider
    ),
    multiMessageRelayer: new Contract(
      l1ContractData.OVM_L1MultiMessageRelayer.abi,
      l1ContractData.OVM_L1MultiMessageRelayer.address,
      signerOrProvider
    ),
    safetyChecker: new Contract(
      l1ContractData.OVM_SafetyChecker.abi,
      l1ContractData.OVM_SafetyChecker.address,
      signerOrProvider
    ),
    stateCommitmentChain: new Contract(
      l1ContractData.OVM_StateCommitmentChain.abi,
      l1ContractData.OVM_StateCommitmentChain.address,
      signerOrProvider
    ),
    stateManagerFactory: new Contract(
      l1ContractData.OVM_StateManagerFactory.abi,
      l1ContractData.OVM_StateManagerFactory.address,
      signerOrProvider
    ),
    stateTransitionerFactory: new Contract(
      l1ContractData.OVM_StateTransitionerFactory.abi,
      l1ContractData.OVM_StateTransitionerFactory.address,
      signerOrProvider
    ),
    xDomainMessengerProxy: new Contract(
      l1ContractData.Proxy__OVM_L1CrossDomainMessenger.abi,
      l1ContractData.Proxy__OVM_L1CrossDomainMessenger.address,
      signerOrProvider
    ),
    l1EthGatewayProxy: new Contract(
      l1ContractData.Proxy__OVM_L1ETHGateway.abi,
      l1ContractData.Proxy__OVM_L1ETHGateway.address,
      signerOrProvider
    ),
    mockBondManger: new Contract(
      l1ContractData.mockOVM_BondManager.abi,
      l1ContractData.mockOVM_BondManager.address,
      signerOrProvider
    ),
  }
}

export const connectL2Contracts = async (
  signerOrProvider: Signer | providers.Provider
): Promise<Object> => {
  const l2ContractData = getL2ContractData()

  return {
    eth: new Contract(
      l2ContractData.OVM_ETH.abi,
      l2ContractData.OVM_ETH.address,
      signerOrProvider
    ),
    xDomainMessenger: new Contract(
      l2ContractData.OVM_L2CrossDomainMessenger.abi,
      l2ContractData.OVM_L2CrossDomainMessenger.address,
      signerOrProvider
    ),
    messagePasser: new Contract(
      l2ContractData.OVM_L2ToL1MessagePasser.abi,
      l2ContractData.OVM_L2ToL1MessagePasser.address,
      signerOrProvider
    ),
    messageSender: new Contract(
      l2ContractData.OVM_L1MessageSender.abi,
      l2ContractData.OVM_L1MessageSender.address,
      signerOrProvider
    ),
    deployerWhiteList: new Contract(
      l2ContractData.OVM_DeployerWhitelist.abi,
      l2ContractData.OVM_DeployerWhitelist.address,
      signerOrProvider
    ),
    ecdsaContractAccount: new Contract(
      l2ContractData.OVM_ECDSAContractAccount.abi,
      l2ContractData.OVM_ECDSAContractAccount.address,
      signerOrProvider
    ),
    sequencerEntrypoint: new Contract(
      l2ContractData.OVM_SequencerEntrypoint.abi,
      l2ContractData.OVM_SequencerEntrypoint.address,
      signerOrProvider
    ),
    erc1820Registry: new Contract(
      l2ContractData.ERC1820Registry.abi,
      l2ContractData.ERC1820Registry.address,
      signerOrProvider
    ),
    addressManager: new Contract(
      l2ContractData.Lib_AddressManager.abi,
      l2ContractData.Lib_AddressManager.address,
      signerOrProvider
    ),
  }
}
