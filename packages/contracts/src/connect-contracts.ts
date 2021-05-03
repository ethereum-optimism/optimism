import { Signer, Contract, providers } from 'ethers'
import { getL1ContractData, getL2ContractData } from './contract-data'

export const connectL1Contracts = async (
  signerOrProvider: Signer | providers.Provider,
  network: 'goerli' | 'kovan' | 'mainnet'
): Promise<{ [key: string]: Contract }> => {
  const l1ContractData = getL1ContractData(network)

  const toEthersContract = (address, abi) =>
    new Contract(address, abi, signerOrProvider)

  return {
    addressManager: toEthersContract(
      l1ContractData.Lib_AddressManager.address,
      l1ContractData.Lib_AddressManager.abi
    ),
    canonicalTransactionChain: toEthersContract(
      l1ContractData.OVM_CanonicalTransactionChain.address,
      l1ContractData.OVM_CanonicalTransactionChain.abi
    ),
    executionManager: toEthersContract(
      l1ContractData.OVM_ExecutionManager.address,
      l1ContractData.OVM_ExecutionManager.abi
    ),
    fraudVerifier: toEthersContract(
      l1ContractData.OVM_FraudVerifier.address,
      l1ContractData.OVM_FraudVerifier.abi
    ),
    xDomainMessenger: toEthersContract(
      l1ContractData.OVM_L1CrossDomainMessenger.address,
      l1ContractData.OVM_L1CrossDomainMessenger.abi
    ),
    ethGateway: toEthersContract(
      l1ContractData.OVM_L1ETHGateway.address,
      l1ContractData.OVM_L1ETHGateway.abi
    ),
    multiMessageRelayer: toEthersContract(
      l1ContractData.OVM_L1MultiMessageRelayer.address,
      l1ContractData.OVM_L1MultiMessageRelayer.abi
    ),
    safetyChecker: toEthersContract(
      l1ContractData.OVM_SafetyChecker.address,
      l1ContractData.OVM_SafetyChecker.abi
    ),
    stateCommitmentChain: toEthersContract(
      l1ContractData.OVM_StateCommitmentChain.address,
      l1ContractData.OVM_StateCommitmentChain.abi
    ),
    stateManagerFactory: toEthersContract(
      l1ContractData.OVM_StateManagerFactory.address,
      l1ContractData.OVM_StateManagerFactory.abi
    ),
    stateTransitionerFactory: toEthersContract(
      l1ContractData.OVM_StateTransitionerFactory.address,
      l1ContractData.OVM_StateTransitionerFactory.abi
    ),
    xDomainMessengerProxy: toEthersContract(
      l1ContractData.Proxy__OVM_L1CrossDomainMessenger.address,
      l1ContractData.Proxy__OVM_L1CrossDomainMessenger.abi
    ),
    l1EthGatewayProxy: toEthersContract(
      l1ContractData.Proxy__OVM_L1ETHGateway.address,
      l1ContractData.Proxy__OVM_L1ETHGateway.abi
    ),
    mockBondManger: toEthersContract(
      l1ContractData.mockOVM_BondManager.address,
      l1ContractData.mockOVM_BondManager.abi
    ),
  }
}

export const connectL2Contracts = async (
  signerOrProvider: Signer | providers.Provider
): Promise<{ [key: string]: Contract }> => {
  const l2ContractData = await getL2ContractData()

  const toEthersContract = (address, abi) =>
    new Contract(address, abi, signerOrProvider)

  return {
    eth: toEthersContract(
      l2ContractData.OVM_ETH.address,
      l2ContractData.OVM_ETH.abi
    ),
    xDomainMessenger: toEthersContract(
      l2ContractData.OVM_L2CrossDomainMessenger.address,
      l2ContractData.OVM_L2CrossDomainMessenger.abi
    ),
    messagePasser: toEthersContract(
      l2ContractData.OVM_L2ToL1MessagePasser.address,
      l2ContractData.OVM_L2ToL1MessagePasser.abi
    ),
    messageSender: toEthersContract(
      l2ContractData.OVM_L1MessageSender.address,
      l2ContractData.OVM_L1MessageSender.abi
    ),
    deployerWhiteList: toEthersContract(
      l2ContractData.OVM_DeployerWhitelist.address,
      l2ContractData.OVM_DeployerWhitelist.abi
    ),
    ecdsaContractAccount: toEthersContract(
      l2ContractData.OVM_ECDSAContractAccount.address,
      l2ContractData.OVM_ECDSAContractAccount.abi
    ),
    sequencerEntrypoint: toEthersContract(
      l2ContractData.OVM_SequencerEntrypoint.address,
      l2ContractData.OVM_SequencerEntrypoint.abi
    ),
    erc1820Registry: toEthersContract(
      l2ContractData.ERC1820Registry.address,
      l2ContractData.ERC1820Registry.abi
    ),
    addressManager: toEthersContract(
      l2ContractData.Lib_AddressManager.address,
      l2ContractData.Lib_AddressManager.abi
    ),
  }
}
