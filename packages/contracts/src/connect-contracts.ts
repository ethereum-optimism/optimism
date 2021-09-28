import { Signer, Contract } from 'ethers'
import { Provider } from '@ethersproject/abstract-provider'
import { getContractArtifact } from './contract-artifacts'
import { getDeployedContractArtifact } from './contract-deployed-artifacts'
import { predeploys } from './predeploys'

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

  const getEthersContract = (name: string) => {
    const artifact = getDeployedContractArtifact(name, network)
    return new Contract(artifact.address, artifact.abi, signerOrProvider)
  }

  return {
    addressManager: getEthersContract('Lib_AddressManager'),
    canonicalTransactionChain: getEthersContract('CanonicalTransactionChain'),
    stateCommitmentChain: getEthersContract('StateCommitmentChain'),
    xDomainMessengerProxy: getEthersContract('Proxy__L1CrossDomainMessenger'),
    bondManager: getEthersContract('mockBondManager'),
  }
}

/**
 * Connects a signer/provider to layer 2 contracts (network agnostic)
 *
 * @param signerOrProvider ethers signer or provider
 * @returns l2 contracts connected to signer/provider
 */
export const connectL2Contracts = async (
  signerOrProvider: any
): Promise<L2Contracts> => {
  checkSignerType(signerOrProvider)

  const getEthersContract = (name: string, iface?: string) => {
    const artifact = getContractArtifact(iface || name)
    const address = predeploys[name]
    return new Contract(address, artifact.abi, signerOrProvider)
  }

  return {
    eth: getEthersContract('OVM_ETH'),
    xDomainMessenger: getEthersContract('L2CrossDomainMessenger'),
    messagePasser: getEthersContract('OVM_L2ToL1MessagePasser'),
    deployerWhiteList: getEthersContract('OVM_DeployerWhitelist'),
  }
}
