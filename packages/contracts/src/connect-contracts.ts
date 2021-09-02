import { Signer, Contract } from 'ethers'
import { Provider } from '@ethersproject/abstract-provider'
import { getL1ContractData, getL2ContractData } from './contract-data'

export type Network = 'goerli' | 'kovan' | 'mainnet'
interface L1Contracts {
  addressManager: Contract
  canonicalTransactionChain: Contract
  stateCommitmentChain: Contract
  xDomainMessengerProxy: Contract
  bondManager: Contract
}

interface L2Contracts {
  eth: Contract
  xDomainMessenger: Contract
  messagePasser: Contract
  messageSender: Contract
  deployerWhiteList: Contract
}

/**
 * Validates user provided a singer or provider & throws error if not
 *
 * @param signerOrProvider
 */
const checkSignerType = (signerOrProvider: Signer | Provider) => {
  if (!signerOrProvider) {
    throw Error('signerOrProvider argument is undefined')
  }
  if (
    !Provider.isProvider(signerOrProvider) &&
    !Signer.isSigner(signerOrProvider)
  ) {
    throw Error('signerOrProvider argument is the wrong type')
  }
}

/**
 * Connects a signer/provider to layer 1 contracts on a given network
 *
 * @param signerOrProvider ethers signer or provider
 * @param network string denoting network
 * @returns l1 contracts connected to signer/provider
 */
export const connectL1Contracts = async (
  signerOrProvider: Signer | Provider,
  network: Network
): Promise<L1Contracts> => {
  checkSignerType(signerOrProvider)

  if (!['mainnet', 'kovan', 'goerli'].includes(network)) {
    throw Error('Must specify network: mainnet, kovan, or goerli.')
  }

  const l1ContractData = getL1ContractData(network)

  const toEthersContract = (data) =>
    new Contract(data.address, data.abi, signerOrProvider)

  return {
    addressManager: toEthersContract(l1ContractData.Lib_AddressManager),
    canonicalTransactionChain: toEthersContract(
      l1ContractData.OVM_CanonicalTransactionChain
    ),
    stateCommitmentChain: toEthersContract(
      l1ContractData.OVM_StateCommitmentChain
    ),
    xDomainMessengerProxy: toEthersContract(
      l1ContractData.Proxy__OVM_L1CrossDomainMessenger
    ),
    bondManager: toEthersContract(l1ContractData.OVM_BondManager),
  }
}

/**
 * Connects a signer/provider to layer 2 contracts (network agnostic)
 *
 * @param signerOrProvider ethers signer or provider
 * @returns l2 contracts connected to signer/provider
 */
export const connectL2Contracts = async (
  signerOrProvider
): Promise<L2Contracts> => {
  const l2ContractData = await getL2ContractData()
  checkSignerType(signerOrProvider)

  const toEthersContract = (data) =>
    new Contract(data.address, data.abi, signerOrProvider)

  return {
    eth: toEthersContract(l2ContractData.OVM_ETH),
    xDomainMessenger: toEthersContract(
      l2ContractData.OVM_L2CrossDomainMessenger
    ),
    messagePasser: toEthersContract(l2ContractData.OVM_L2ToL1MessagePasser),
    messageSender: toEthersContract(l2ContractData.OVM_L1MessageSender),
    deployerWhiteList: toEthersContract(l2ContractData.OVM_DeployerWhitelist),
  }
}
