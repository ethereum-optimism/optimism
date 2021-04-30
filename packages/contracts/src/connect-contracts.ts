import { Signer, Contract, providers } from 'ethers'
import { getL1ContractData, getL2ContractData } from './contract-data'

export const connectL1Contracts = async (
  signerOrProvider: Signer | providers.Provider,
  network: 'goerli' | 'kovan' | 'mainnet'
): Promise<Object> => {
  const l1ContractData = getL1ContractData(network)

  return {
    addressManager: new Contract(
      l1ContractData.Lib_AddressManager.address,
      l1ContractData.Lib_AddressManager.abi,
      signerOrProvider
    ),
    canonicalTransactionChain: new Contract(
      l1ContractData.OVM_CanonicalTransactionChain.address,
      l1ContractData.OVM_CanonicalTransactionChain.abi,
      signerOrProvider
    ),
    executionManager: new Contract(
      l1ContractData.OVM_ExecutionManager.address,
      l1ContractData.OVM_ExecutionManager.abi,
      signerOrProvider
    ),
    fraudVerifier: new Contract(
      l1ContractData.OVM_FraudVerifier.address,
      l1ContractData.OVM_FraudVerifier.abi,
      signerOrProvider
    ),
    xDomainMessenger: new Contract(
      l1ContractData.OVM_L1CrossDomainMessenger.address,
      l1ContractData.OVM_L1CrossDomainMessenger.abi,
      signerOrProvider
    ),
    ethGateway: new Contract(
      l1ContractData.OVM_L1ETHGateway.address,
      l1ContractData.OVM_L1ETHGateway.abi,
      signerOrProvider
    ),
    multiMessageRelayer: new Contract(
      l1ContractData.OVM_L1MultiMessageRelayer.address,
      l1ContractData.OVM_L1MultiMessageRelayer.abi,
      signerOrProvider
    ),
    safetyChecker: new Contract(
      l1ContractData.OVM_SafetyChecker.address,
      l1ContractData.OVM_SafetyChecker.abi,
      signerOrProvider
    ),
    stateCommitmentChain: new Contract(
      l1ContractData.OVM_StateCommitmentChain.address,
      l1ContractData.OVM_StateCommitmentChain.abi,
      signerOrProvider
    ),
    stateManagerFactory: new Contract(
      l1ContractData.OVM_StateManagerFactory.address,
      l1ContractData.OVM_StateManagerFactory.abi,
      signerOrProvider
    ),
    stateTransitionerFactory: new Contract(
      l1ContractData.OVM_StateTransitionerFactory.address,
      l1ContractData.OVM_StateTransitionerFactory.abi,
      signerOrProvider
    ),
    xDomainMessengerProxy: new Contract(
      l1ContractData.Proxy__OVM_L1CrossDomainMessenger.address,
      l1ContractData.Proxy__OVM_L1CrossDomainMessenger.abi,
      signerOrProvider
    ),
    l1EthGatewayProxy: new Contract(
      l1ContractData.Proxy__OVM_L1ETHGateway.address,
      l1ContractData.Proxy__OVM_L1ETHGateway.abi,
      signerOrProvider
    ),
    mockBondManger: new Contract(
      l1ContractData.mockOVM_BondManager.address,
      l1ContractData.mockOVM_BondManager.abi,
      signerOrProvider
    ),
  }
}

export const connectL2Contracts = async (
  signerOrProvider: Signer | providers.Provider
): Promise<Object> => {
  const l2ContractData = await getL2ContractData()

  return {
    eth: new Contract(
      l2ContractData.OVM_ETH.abi,
      l2ContractData.OVM_ETH.address,
      signerOrProvider
    ),
    xDomainMessenger: new Contract(
      l2ContractData.OVM_L2CrossDomainMessenger.address,
      l2ContractData.OVM_L2CrossDomainMessenger.abi,
      signerOrProvider
    ),
    messagePasser: new Contract(
      l2ContractData.OVM_L2ToL1MessagePasser.address,
      l2ContractData.OVM_L2ToL1MessagePasser.abi,
      signerOrProvider
    ),
    messageSender: new Contract(
      l2ContractData.OVM_L1MessageSender.address,
      l2ContractData.OVM_L1MessageSender.abi,
      signerOrProvider
    ),
    deployerWhiteList: new Contract(
      l2ContractData.OVM_DeployerWhitelist.address,
      l2ContractData.OVM_DeployerWhitelist.abi,
      signerOrProvider
    ),
    ecdsaContractAccount: new Contract(
      l2ContractData.OVM_ECDSAContractAccount.address,
      l2ContractData.OVM_ECDSAContractAccount.abi,
      signerOrProvider
    ),
    sequencerEntrypoint: new Contract(
      l2ContractData.OVM_SequencerEntrypoint.address,
      l2ContractData.OVM_SequencerEntrypoint.abi,
      signerOrProvider
    ),
    erc1820Registry: new Contract(
      l2ContractData.ERC1820Registry.address,
      l2ContractData.ERC1820Registry.abi,
      signerOrProvider
    ),
    addressManager: new Contract(
      l2ContractData.Lib_AddressManager.address,
      l2ContractData.Lib_AddressManager.abi,
      signerOrProvider
    ),
  }
}
