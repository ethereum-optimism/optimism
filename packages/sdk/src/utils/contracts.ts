import { getContractInterface, predeploys } from '@eth-optimism/contracts'
import { ethers, Contract } from 'ethers'

import { toAddress } from './coercion'
import { DeepPartial } from './type-utils'
import {
  OEContracts,
  OEL1Contracts,
  OEL2Contracts,
  OEContractsLike,
  OEL2ContractsLike,
  AddressLike,
  BridgeAdapters,
  BridgeAdapterData,
  ICrossChainMessenger,
} from '../interfaces'
import {
  StandardBridgeAdapter,
  ETHBridgeAdapter,
  DAIBridgeAdapter,
} from '../adapters'

/**
 * Full list of default L2 contract addresses.
 */
export const DEFAULT_L2_CONTRACT_ADDRESSES: OEL2ContractsLike = {
  L2CrossDomainMessenger: predeploys.L2CrossDomainMessenger,
  L2StandardBridge: predeploys.L2StandardBridge,
  OVM_L1BlockNumber: predeploys.OVM_L1BlockNumber,
  OVM_L2ToL1MessagePasser: predeploys.OVM_L2ToL1MessagePasser,
  OVM_DeployerWhitelist: predeploys.OVM_DeployerWhitelist,
  OVM_ETH: predeploys.OVM_ETH,
  OVM_GasPriceOracle: predeploys.OVM_GasPriceOracle,
  OVM_SequencerFeeVault: predeploys.OVM_SequencerFeeVault,
  WETH: predeploys.WETH9,
}

/**
 * We've changed some contract names in this SDK to be a bit nicer. Here we remap these nicer names
 * back to the original contract names so we can look them up.
 */
const NAME_REMAPPING = {
  AddressManager: 'Lib_AddressManager',
  OVM_L1BlockNumber: 'iOVM_L1BlockNumber',
  WETH: 'WETH9',
}

/**
 * Mapping of L1 chain IDs to the appropriate contract addresses for the OE deployments to the
 * given network. Simplifies the process of getting the correct contract addresses for a given
 * contract name.
 */
export const CONTRACT_ADDRESSES: {
  [l1ChainId: number]: OEContractsLike
} = {
  // Mainnet
  1: {
    l1: {
      AddressManager: '0xdE1FCfB0851916CA5101820A69b13a4E276bd81F',
      L1CrossDomainMessenger: '0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1',
      L1StandardBridge: '0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1',
      StateCommitmentChain: '0xBe5dAb4A2e9cd0F27300dB4aB94BeE3A233AEB19',
      CanonicalTransactionChain: '0x5E4e65926BA27467555EB562121fac00D24E9dD2',
      BondManager: '0xcd626E1328b41fCF24737F137BcD4CE0c32bc8d1',
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  // Kovan
  42: {
    l1: {
      AddressManager: '0x100Dd3b414Df5BbA2B542864fF94aF8024aFdf3a',
      L1CrossDomainMessenger: '0x4361d0F75A0186C05f971c566dC6bEa5957483fD',
      L1StandardBridge: '0x22F24361D548e5FaAfb36d1437839f080363982B',
      StateCommitmentChain: '0xD7754711773489F31A0602635f3F167826ce53C5',
      CanonicalTransactionChain: '0xf7B88A133202d41Fe5E2Ab22e6309a1A4D50AF74',
      BondManager: '0xc5a603d273E28185c18Ba4d26A0024B2d2F42740',
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  // Goerli
  5: {
    l1: {
      AddressManager: '0x2F7E3cAC91b5148d336BbffB224B4dC79F09f01D',
      L1CrossDomainMessenger: '0xEcC89b9EDD804850C4F343A278Be902be11AaF42',
      L1StandardBridge: '0x73298186A143a54c20ae98EEE5a025bD5979De02',
      StateCommitmentChain: '0x1afcA918eff169eE20fF8AB6Be75f3E872eE1C1A',
      CanonicalTransactionChain: '0x2ebA8c4EfDB39A8Cd8f9eD65c50ec079f7CEBD81',
      BondManager: '0xE5AE60bD6F8DEe4D0c2BC9268e23B92F1cacC58F',
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  // Hardhat local
  31337: {
    l1: {
      AddressManager: '0x5FbDB2315678afecb367f032d93F642f64180aa3',
      L1CrossDomainMessenger: '0x8A791620dd6260079BF849Dc5567aDC3F2FdC318',
      L1StandardBridge: '0x610178dA211FEF7D417bC0e6FeD39F05609AD788',
      StateCommitmentChain: '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9',
      CanonicalTransactionChain: '0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9',
      BondManager: '0x5FC8d32690cc91D4c39d9d3abcBD16989F875707',
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
}

/**
 * Mapping of L1 chain IDs to the list of custom bridge addresses for each chain.
 */
export const BRIDGE_ADAPTER_DATA: {
  [l1ChainId: number]: BridgeAdapterData
} = {
  // TODO: Maybe we can pull these automatically from the token list?
  // Alternatively, check against the token list in CI.
  1: {
    Standard: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: CONTRACT_ADDRESSES[1].l1.L1StandardBridge,
      l2Bridge: predeploys.L2StandardBridge,
    },
    ETH: {
      Adapter: ETHBridgeAdapter,
      l1Bridge: CONTRACT_ADDRESSES[1].l1.L1StandardBridge,
      l2Bridge: predeploys.L2StandardBridge,
    },
    BitBTC: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: '0xaBA2c5F108F7E820C049D5Af70B16ac266c8f128',
      l2Bridge: '0x158F513096923fF2d3aab2BcF4478536de6725e2',
    },
    DAI: {
      Adapter: DAIBridgeAdapter,
      l1Bridge: '0x10E6593CDda8c58a1d0f14C5164B376352a55f2F',
      l2Bridge: '0x467194771dAe2967Aef3ECbEDD3Bf9a310C76C65',
    },
  },
  42: {
    Standard: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: CONTRACT_ADDRESSES[42].l1.L1StandardBridge,
      l2Bridge: predeploys.L2StandardBridge,
    },
    ETH: {
      Adapter: ETHBridgeAdapter,
      l1Bridge: CONTRACT_ADDRESSES[42].l1.L1StandardBridge,
      l2Bridge: predeploys.L2StandardBridge,
    },
    BitBTC: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: '0x0b651A42F32069d62d5ECf4f2a7e5Bd3E9438746',
      l2Bridge: '0x0CFb46528a7002a7D8877a5F7a69b9AaF1A9058e',
    },
    USX: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: '0x40E862341b2416345F02c41Ac70df08525150dC7',
      l2Bridge: '0xB4d37826b14Cd3CB7257A2A5094507d701fe715f',
    },
    DAI: {
      Adapter: DAIBridgeAdapter,
      l1Bridge: '0xb415e822C4983ecD6B1c1596e8a5f976cf6CD9e3',
      l2Bridge: '0x467194771dAe2967Aef3ECbEDD3Bf9a310C76C65',
    },
  },
  5: {
    Standard: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: CONTRACT_ADDRESSES[5].l1.L1StandardBridge,
      l2Bridge: predeploys.L2StandardBridge,
    },
    ETH: {
      Adapter: ETHBridgeAdapter,
      l1Bridge: CONTRACT_ADDRESSES[5].l1.L1StandardBridge,
      l2Bridge: predeploys.L2StandardBridge,
    },
  },
  31337: {
    Standard: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: CONTRACT_ADDRESSES[31337].l1.L1StandardBridge,
      l2Bridge: predeploys.L2StandardBridge,
    },
    ETH: {
      Adapter: ETHBridgeAdapter,
      l1Bridge: CONTRACT_ADDRESSES[31337].l1.L1StandardBridge,
      l2Bridge: predeploys.L2StandardBridge,
    },
  },
}

// TODO: PR is big enough as-is, will add support for SNX in another PR
// MAINNET
// l1: {
//   SNX: '0xCd9D4988C0AE61887B075bA77f08cbFAd2b65068',
// },
// l2: {
//   SNX: '0x3f87Ff1de58128eF8FCb4c807eFD776E1aC72E51',
// },
// KOVAN
// l1: {
//   SNX: '0xD134Db47DDF5A6feB245452af17cCAf92ee53D3c',
// },
// l2: {
//   SNX: '0x5C3f51CEd0C2F6157e2be67c029264D6C44bfe42',
// },

/**
 * Returns an ethers.Contract object for the given name, connected to the appropriate address for
 * the given L1 chain ID. Users can also provide a custom address to connect the contract to
 * instead. If the chain ID is not known then the user MUST provide a custom address or this
 * function will throw an error.
 *
 * @param contractName Name of the contract to connect to.
 * @param l1ChainId Chain ID for the L1 network where the OE contracts are deployed.
 * @param opts Additional options for connecting to the contract.
 * @param opts.address Custom address to connect to the contract.
 * @param opts.signerOrProvider Signer or provider to connect to the contract.
 * @returns An ethers.Contract object connected to the appropriate address and interface.
 */
export const getOEContract = (
  contractName: keyof OEL1Contracts | keyof OEL2Contracts,
  l1ChainId: number,
  opts: {
    address?: AddressLike
    signerOrProvider?: ethers.Signer | ethers.providers.Provider
  } = {}
): Contract => {
  const addresses = CONTRACT_ADDRESSES[l1ChainId]
  if (addresses === undefined && opts.address === undefined) {
    throw new Error(
      `cannot get contract ${contractName} for unknown L1 chain ID ${l1ChainId}, you must provide an address`
    )
  }

  return new Contract(
    toAddress(
      opts.address || addresses.l1[contractName] || addresses.l2[contractName]
    ),
    getContractInterface(NAME_REMAPPING[contractName] || contractName),
    opts.signerOrProvider
  )
}

/**
 * Automatically connects to all contract addresses, both L1 and L2, for the given L1 chain ID. The
 * user can provide custom contract address overrides for L1 or L2 contracts. If the given chain ID
 * is not known then the user MUST provide custom contract addresses for ALL L1 contracts or this
 * function will throw an error.
 *
 * @param l1ChainId Chain ID for the L1 network where the OE contracts are deployed.
 * @param opts Additional options for connecting to the contracts.
 * @param opts.l1SignerOrProvider: Signer or provider to connect to the L1 contracts.
 * @param opts.l2SignerOrProvider: Signer or provider to connect to the L2 contracts.
 * @param opts.overrides Custom contract address overrides for L1 or L2 contracts.
 * @returns An object containing ethers.Contract objects connected to the appropriate addresses on
 * both L1 and L2.
 */
export const getAllOEContracts = (
  l1ChainId: number,
  opts: {
    l1SignerOrProvider?: ethers.Signer | ethers.providers.Provider
    l2SignerOrProvider?: ethers.Signer | ethers.providers.Provider
    overrides?: DeepPartial<OEContractsLike>
  } = {}
): OEContracts => {
  const addresses = CONTRACT_ADDRESSES[l1ChainId] || {
    l1: {
      AddressManager: undefined,
      L1CrossDomainMessenger: undefined,
      L1StandardBridge: undefined,
      StateCommitmentChain: undefined,
      CanonicalTransactionChain: undefined,
      BondManager: undefined,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  }

  // Attach all L1 contracts.
  const l1Contracts: OEL1Contracts = {} as any
  for (const [contractName, contractAddress] of Object.entries(addresses.l1)) {
    l1Contracts[contractName] = getOEContract(contractName as any, l1ChainId, {
      address: opts.overrides?.l1?.[contractName] || contractAddress,
      signerOrProvider: opts.l1SignerOrProvider,
    })
  }

  // Attach all L2 contracts.
  const l2Contracts: OEL2Contracts = {} as any
  for (const [contractName, contractAddress] of Object.entries(addresses.l2)) {
    l2Contracts[contractName] = getOEContract(contractName as any, l1ChainId, {
      address: opts.overrides?.l2?.[contractName] || contractAddress,
      signerOrProvider: opts.l2SignerOrProvider,
    })
  }

  return {
    l1: l1Contracts,
    l2: l2Contracts,
  }
}

/**
 * Gets a series of bridge adapters for the given L1 chain ID.
 *
 * @param l1ChainId L1 chain ID for the L1 network where the custom bridges are deployed.
 * @param messenger Cross chain messenger to connect to the bridge adapters
 * @param opts Additional options for connecting to the custom bridges.
 * @param opts.overrides Custom bridge adapters.
 * @returns An object containing all bridge adapters
 */
export const getBridgeAdapters = (
  l1ChainId: number,
  messenger: ICrossChainMessenger,
  opts?: {
    overrides?: BridgeAdapterData
  }
): BridgeAdapters => {
  const adapters: BridgeAdapters = {}
  for (const [bridgeName, bridgeData] of Object.entries({
    ...(BRIDGE_ADAPTER_DATA[l1ChainId] || {}),
    ...(opts?.overrides || {}),
  })) {
    adapters[bridgeName] = new bridgeData.Adapter({
      messenger,
      l1Bridge: bridgeData.l1Bridge,
      l2Bridge: bridgeData.l2Bridge,
    })
  }

  return adapters
}
