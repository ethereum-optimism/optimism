import { getContractInterface, predeploys } from '@eth-optimism/contracts'
import { ethers, Contract } from 'ethers'
import l1StandardBridge from '@eth-optimism/contracts-bedrock/forge-artifacts/L1StandardBridge.sol/L1StandardBridge.json'
import l2StandardBridge from '@eth-optimism/contracts-bedrock/forge-artifacts/L2StandardBridge.sol/L2StandardBridge.json'
import optimismMintableERC20 from '@eth-optimism/contracts-bedrock/forge-artifacts/OptimismMintableERC20.sol/OptimismMintableERC20.json'
import optimismPortal from '@eth-optimism/contracts-bedrock/forge-artifacts/OptimismPortal.sol/OptimismPortal.json'
import l1CrossDomainMessenger from '@eth-optimism/contracts-bedrock/forge-artifacts/L1CrossDomainMessenger.sol/L1CrossDomainMessenger.json'
import l2CrossDomainMessenger from '@eth-optimism/contracts-bedrock/forge-artifacts/L2CrossDomainMessenger.sol/L2CrossDomainMessenger.json'
import optimismMintableERC20Factory from '@eth-optimism/contracts-bedrock/forge-artifacts/OptimismMintableERC20Factory.sol/OptimismMintableERC20Factory.json'
import proxyAdmin from '@eth-optimism/contracts-bedrock/forge-artifacts/ProxyAdmin.sol/ProxyAdmin.json'
import l2OutputOracle from '@eth-optimism/contracts-bedrock/forge-artifacts/L2OutputOracle.sol/L2OutputOracle.json'
import l1ERC721Bridge from '@eth-optimism/contracts-bedrock/forge-artifacts/L1ERC721Bridge.sol/L1ERC721Bridge.json'
import l2ERC721Bridge from '@eth-optimism/contracts-bedrock/forge-artifacts/L2ERC721Bridge.sol/L2ERC721Bridge.json'
import l1Block from '@eth-optimism/contracts-bedrock/forge-artifacts/L1Block.sol/L1Block.json'
import l2ToL1MessagePasser from '@eth-optimism/contracts-bedrock/forge-artifacts/L2ToL1MessagePasser.sol/L2ToL1MessagePasser.json'
import gasPriceOracle from '@eth-optimism/contracts-bedrock/forge-artifacts/GasPriceOracle.sol/GasPriceOracle.json'

import { toAddress } from './coercion'
import { DeepPartial } from './type-utils'
import { CrossChainMessenger } from '../cross-chain-messenger'
import { StandardBridgeAdapter, ETHBridgeAdapter } from '../adapters'
import {
  CONTRACT_ADDRESSES,
  DEFAULT_L2_CONTRACT_ADDRESSES,
  BRIDGE_ADAPTER_DATA,
} from './chain-constants'
import {
  OEContracts,
  OEL1Contracts,
  OEL2Contracts,
  OEContractsLike,
  AddressLike,
  BridgeAdapters,
  BridgeAdapterData,
} from '../interfaces'

/**
 * We've changed some contract names in this SDK to be a bit nicer. Here we remap these nicer names
 * back to the original contract names so we can look them up.
 */
const NAME_REMAPPING = {
  AddressManager: 'Lib_AddressManager' as const,
  OVM_L1BlockNumber: 'iOVM_L1BlockNumber' as const,
  WETH: 'WETH9' as const,
  BedrockMessagePasser: 'L2ToL1MessagePasser' as const,
}

const getContractInterfaceBedrock = (name: string): ethers.utils.Interface => {
  let artifact: any = ''
  switch (name) {
    case 'Lib_AddressManager':
    case 'AddressManager':
      artifact = ''
      break
    case 'L1CrossDomainMessenger':
      artifact = l1CrossDomainMessenger
      break
    case 'L1ERC721Bridge':
      artifact = l1ERC721Bridge
      break
    case 'L2OutputOracle':
      artifact = l2OutputOracle
      break
    case 'OptimismMintableERC20Factory':
      artifact = optimismMintableERC20Factory
      break
    case 'ProxyAdmin':
      artifact = proxyAdmin
      break
    case 'L1StandardBridge':
      artifact = l1StandardBridge
      break
    case 'L2StandardBridge':
      artifact = l2StandardBridge
      break
    case 'OptimismPortal':
      artifact = optimismPortal
      break
    case 'L2CrossDomainMessenger':
      artifact = l2CrossDomainMessenger
      break
    case 'OptimismMintableERC20':
      artifact = optimismMintableERC20
      break
    case 'L2ERC721Bridge':
      artifact = l2ERC721Bridge
      break
    case 'L1Block':
      artifact = l1Block
      break
    case 'L2ToL1MessagePasser':
      artifact = l2ToL1MessagePasser
      break
    case 'GasPriceOracle':
      artifact = gasPriceOracle
      break
  }
  return new ethers.utils.Interface(artifact.abi)
}

/**
 * Returns an ethers.Contract object for the given name, connected to the appropriate address for
 * the given L2 chain ID. Users can also provide a custom address to connect the contract to
 * instead. If the chain ID is not known then the user MUST provide a custom address or this
 * function will throw an error.
 *
 * @param contractName Name of the contract to connect to.
 * @param l2ChainId Chain ID for the L2 network.
 * @param opts Additional options for connecting to the contract.
 * @param opts.address Custom address to connect to the contract.
 * @param opts.signerOrProvider Signer or provider to connect to the contract.
 * @returns An ethers.Contract object connected to the appropriate address and interface.
 */
export const getOEContract = (
  contractName: keyof OEL1Contracts | keyof OEL2Contracts,
  l2ChainId: number,
  opts: {
    address?: AddressLike
    signerOrProvider?: ethers.Signer | ethers.providers.Provider
  } = {}
): Contract => {
  const addresses = CONTRACT_ADDRESSES[l2ChainId]
  if (addresses === undefined && opts.address === undefined) {
    throw new Error(
      `cannot get contract ${contractName} for unknown L2 chain ID ${l2ChainId}, you must provide an address`
    )
  }

  // Bedrock interfaces are backwards compatible. We can prefer Bedrock interfaces over legacy
  // interfaces if they exist.
  const name = NAME_REMAPPING[contractName] || contractName
  let iface: ethers.utils.Interface
  try {
    iface = getContractInterfaceBedrock(name)
  } catch (err) {
    iface = getContractInterface(name)
  }

  return new Contract(
    toAddress(
      opts.address || addresses.l1[contractName] || addresses.l2[contractName]
    ),
    iface,
    opts.signerOrProvider
  )
}

/**
 * Automatically connects to all contract addresses, both L1 and L2, for the given L2 chain ID. The
 * user can provide custom contract address overrides for L1 or L2 contracts. If the given chain ID
 * is not known then the user MUST provide custom contract addresses for ALL L1 contracts or this
 * function will throw an error.
 *
 * @param l2ChainId Chain ID for the L2 network.
 * @param opts Additional options for connecting to the contracts.
 * @param opts.l1SignerOrProvider: Signer or provider to connect to the L1 contracts.
 * @param opts.l2SignerOrProvider: Signer or provider to connect to the L2 contracts.
 * @param opts.overrides Custom contract address overrides for L1 or L2 contracts.
 * @returns An object containing ethers.Contract objects connected to the appropriate addresses on
 * both L1 and L2.
 */
export const getAllOEContracts = (
  l2ChainId: number,
  opts: {
    l1SignerOrProvider?: ethers.Signer | ethers.providers.Provider
    l2SignerOrProvider?: ethers.Signer | ethers.providers.Provider
    overrides?: DeepPartial<OEContractsLike>
  } = {}
): OEContracts => {
  const addresses = CONTRACT_ADDRESSES[l2ChainId] || {
    l1: {
      AddressManager: undefined,
      L1CrossDomainMessenger: undefined,
      L1StandardBridge: undefined,
      StateCommitmentChain: undefined,
      CanonicalTransactionChain: undefined,
      BondManager: undefined,
      OptimismPortal: undefined,
      L2OutputOracle: undefined,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  }

  // Attach all L1 contracts.
  const l1Contracts = {} as OEL1Contracts
  for (const [contractName, contractAddress] of Object.entries(addresses.l1)) {
    l1Contracts[contractName] = getOEContract(
      contractName as keyof OEL1Contracts,
      l2ChainId,
      {
        address: opts.overrides?.l1?.[contractName] || contractAddress,
        signerOrProvider: opts.l1SignerOrProvider,
      }
    )
  }

  // Attach all L2 contracts.
  const l2Contracts = {} as OEL2Contracts
  for (const [contractName, contractAddress] of Object.entries(addresses.l2)) {
    l2Contracts[contractName] = getOEContract(
      contractName as keyof OEL2Contracts,
      l2ChainId,
      {
        address: opts.overrides?.l2?.[contractName] || contractAddress,
        signerOrProvider: opts.l2SignerOrProvider,
      }
    )
  }

  return {
    l1: l1Contracts,
    l2: l2Contracts,
  }
}

/**
 * Gets a series of bridge adapters for the given L2 chain ID.
 *
 * @param l2ChainId Chain ID for the L2 network.
 * @param messenger Cross chain messenger to connect to the bridge adapters
 * @param opts Additional options for connecting to the custom bridges.
 * @param opts.overrides Custom bridge adapters.
 * @returns An object containing all bridge adapters
 */
export const getBridgeAdapters = (
  l2ChainId: number,
  messenger: CrossChainMessenger,
  opts?: {
    overrides?: BridgeAdapterData
    contracts?: DeepPartial<OEContractsLike>
  }
): BridgeAdapters => {
  const adapterData: BridgeAdapterData = {
    ...(CONTRACT_ADDRESSES[l2ChainId] || opts?.contracts?.l1?.L1StandardBridge
      ? {
          Standard: {
            Adapter: StandardBridgeAdapter,
            l1Bridge:
              opts?.contracts?.l1?.L1StandardBridge ||
              CONTRACT_ADDRESSES[l2ChainId].l1.L1StandardBridge,
            l2Bridge: predeploys.L2StandardBridge,
          },
          ETH: {
            Adapter: ETHBridgeAdapter,
            l1Bridge:
              opts?.contracts?.l1?.L1StandardBridge ||
              CONTRACT_ADDRESSES[l2ChainId].l1.L1StandardBridge,
            l2Bridge: predeploys.L2StandardBridge,
          },
        }
      : {}),
    ...(BRIDGE_ADAPTER_DATA[l2ChainId] || {}),
    ...(opts?.overrides || {}),
  }

  const adapters: BridgeAdapters = {}
  for (const [bridgeName, bridgeData] of Object.entries(adapterData)) {
    adapters[bridgeName] = new bridgeData.Adapter({
      messenger,
      l1Bridge: bridgeData.l1Bridge,
      l2Bridge: bridgeData.l2Bridge,
    })
  }

  return adapters
}
