import { predeploys } from '@eth-optimism/core-utils'
import { ethers } from 'ethers'

// The addresses below should be for the proxy if it is a proxied contract.

const portalAddresses = {
  mainnet: '0xbEb5Fc579115071764c7423A4f12eDde41f106Ed',
  goerli: '0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383',
  sepolia: '0x16Fc5058F25648194471939df75CF27A2fdC48BC',
}

const l2OutputOracleAddresses = {
  mainnet: '0xdfe97868233d1aa22e815a266982f2cf17685a27',
  goerli: '0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0',
  sepolia: '0x90E9c4f8a994a250F6aEfd61CAFb4F2e895D458F',
}

const addressManagerAddresses = {
  mainnet: '0xdE1FCfB0851916CA5101820A69b13a4E276bd81F',
  goerli: '0xa6f73589243a6A7a9023b1Fa0651b1d89c177111',
  sepolia: '0x9bFE9c5609311DF1c011c47642253B78a4f33F4B',
}

const l1StandardBridgeAddresses = {
  mainnet: '0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1',
  goerli: '0x636Af16bf2f682dD3109e60102b8E1A089FedAa8',
  sepolia: '0xFBb0621E0B23b5478B630BD55a5f21f67730B0F1',
}

const l1CrossDomainMessengerAddresses = {
  mainnet: '0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1',
  goerli: '0x5086d1eEF304eb5284A0f6720f79403b4e9bE294',
  sepolia: '0x58Cc85b8D04EA49cC6DBd3CbFFd00B4B8D6cb3ef',
}

// legacy
const stateCommitmentChainAddresses = {
  mainnet: '0xBe5dAb4A2e9cd0F27300dB4aB94BeE3A233AEB19',
  goerli: '0x9c945aC97Baf48cB784AbBB61399beB71aF7A378',
  sepolia: ethers.constants.AddressZero,
}

// legacy
const canonicalTransactionChainAddresses = {
  mainnet: '0x5E4e65926BA27467555EB562121fac00D24E9dD2',
  goerli: '0x607F755149cFEB3a14E1Dc3A4E2450Cde7dfb04D',
  sepolia: ethers.constants.AddressZero,
}

import {
  L1ChainID,
  L2ChainID,
  OEContractsLike,
  OEL1ContractsLike,
  OEL2ContractsLike,
  BridgeAdapterData,
} from '../interfaces'
import {
  StandardBridgeAdapter,
  DAIBridgeAdapter,
  ECOBridgeAdapter,
} from '../adapters'

export const DEPOSIT_CONFIRMATION_BLOCKS: {
  [ChainID in L2ChainID]: number
} = {
  [L2ChainID.OPTIMISM]: 50 as const,
  [L2ChainID.OPTIMISM_GOERLI]: 12 as const,
  [L2ChainID.OPTIMISM_SEPOLIA]: 12 as const,
  [L2ChainID.OPTIMISM_HARDHAT_LOCAL]: 2 as const,
  [L2ChainID.OPTIMISM_HARDHAT_DEVNET]: 2 as const,
  [L2ChainID.OPTIMISM_BEDROCK_ALPHA_TESTNET]: 12 as const,
  [L2ChainID.BASE_GOERLI]: 25 as const,
  [L2ChainID.BASE_SEPOLIA]: 25 as const,
  [L2ChainID.BASE_MAINNET]: 10 as const,
  [L2ChainID.ZORA_GOERLI]: 12 as const,
  [L2ChainID.ZORA_MAINNET]: 50 as const,
}

export const CHAIN_BLOCK_TIMES: {
  [ChainID in L1ChainID]: number
} = {
  [L1ChainID.MAINNET]: 13 as const,
  [L1ChainID.GOERLI]: 15 as const,
  [L1ChainID.SEPOLIA]: 15 as const,
  [L1ChainID.HARDHAT_LOCAL]: 1 as const,
  [L1ChainID.BEDROCK_LOCAL_DEVNET]: 15 as const,
}

/**
 * Full list of default L2 contract addresses.
 */
export const DEFAULT_L2_CONTRACT_ADDRESSES: OEL2ContractsLike = {
  L2CrossDomainMessenger: predeploys.L2CrossDomainMessenger,
  L2ToL1MessagePasser: predeploys.L2ToL1MessagePasser,
  L2StandardBridge: predeploys.L2StandardBridge,
  OVM_L1BlockNumber: predeploys.L1BlockNumber,
  OVM_L2ToL1MessagePasser: predeploys.L2ToL1MessagePasser,
  OVM_DeployerWhitelist: predeploys.DeployerWhitelist,
  OVM_ETH: predeploys.LegacyERC20ETH,
  OVM_GasPriceOracle: predeploys.GasPriceOracle,
  OVM_SequencerFeeVault: predeploys.SequencerFeeVault,
  WETH: predeploys.WETH9,
  BedrockMessagePasser: predeploys.L2ToL1MessagePasser,
}

/**
 * Loads the L1 contracts for a given network by the network name.
 *
 * @param network The name of the network to load the contracts for.
 * @returns The L1 contracts for the given network.
 */
const getL1ContractsByNetworkName = (network: string): OEL1ContractsLike => {
  return {
    AddressManager: addressManagerAddresses[network],
    L1CrossDomainMessenger: l1CrossDomainMessengerAddresses[network],
    L1StandardBridge: l1StandardBridgeAddresses[network],
    StateCommitmentChain: stateCommitmentChainAddresses[network],
    CanonicalTransactionChain: canonicalTransactionChainAddresses[network],
    BondManager: ethers.constants.AddressZero,
    OptimismPortal: portalAddresses[network],
    L2OutputOracle: l2OutputOracleAddresses[network],
    OptimismPortal2: portalAddresses[network],
    DisputeGameFactory: ethers.constants.AddressZero,
  }
}

/**
 * List of contracts that are ignorable when checking for contracts on a given network.
 */
export const IGNORABLE_CONTRACTS = ['OptimismPortal2', 'DisputeGameFactory']

/**
 * Mapping of L1 chain IDs to the appropriate contract addresses for the OE deployments to the
 * given network. Simplifies the process of getting the correct contract addresses for a given
 * contract name.
 */
export const CONTRACT_ADDRESSES: {
  [ChainID in L2ChainID]: OEContractsLike
} = {
  [L2ChainID.OPTIMISM]: {
    l1: getL1ContractsByNetworkName('mainnet'),
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.OPTIMISM_GOERLI]: {
    l1: getL1ContractsByNetworkName('goerli'),
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.OPTIMISM_SEPOLIA]: {
    l1: getL1ContractsByNetworkName('sepolia'),
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.OPTIMISM_HARDHAT_LOCAL]: {
    l1: {
      AddressManager: '0x5FbDB2315678afecb367f032d93F642f64180aa3' as const,
      L1CrossDomainMessenger:
        '0x8A791620dd6260079BF849Dc5567aDC3F2FdC318' as const,
      L1StandardBridge: '0x610178dA211FEF7D417bC0e6FeD39F05609AD788' as const,
      StateCommitmentChain:
        '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9' as const,
      CanonicalTransactionChain:
        '0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9' as const,
      BondManager: '0x5FC8d32690cc91D4c39d9d3abcBD16989F875707' as const,
      // FIXME
      OptimismPortal: '0x0000000000000000000000000000000000000000' as const,
      L2OutputOracle: '0x0000000000000000000000000000000000000000' as const,
      OptimismPortal2: '0x0000000000000000000000000000000000000000' as const,
      DisputeGameFactory: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.OPTIMISM_HARDHAT_DEVNET]: {
    l1: {
      AddressManager: '0x5FbDB2315678afecb367f032d93F642f64180aa3' as const,
      L1CrossDomainMessenger:
        '0x8A791620dd6260079BF849Dc5567aDC3F2FdC318' as const,
      L1StandardBridge: '0x610178dA211FEF7D417bC0e6FeD39F05609AD788' as const,
      StateCommitmentChain:
        '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9' as const,
      CanonicalTransactionChain:
        '0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9' as const,
      BondManager: '0x5FC8d32690cc91D4c39d9d3abcBD16989F875707' as const,
      OptimismPortal: '0x0000000000000000000000000000000000000000' as const,
      L2OutputOracle: '0x0000000000000000000000000000000000000000' as const,
      OptimismPortal2: '0x0000000000000000000000000000000000000000' as const,
      DisputeGameFactory: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.OPTIMISM_BEDROCK_ALPHA_TESTNET]: {
    l1: {
      AddressManager: '0xb4e08DcE1F323608229265c9d4125E22a4B9dbAF' as const,
      L1CrossDomainMessenger:
        '0x838a6DC4E37CA45D4Ef05bb776bf05eEf50798De' as const,
      L1StandardBridge: '0xFf94B6C486350aD92561Ba09bad3a59df764Da92' as const,
      StateCommitmentChain:
        '0x0000000000000000000000000000000000000000' as const,
      CanonicalTransactionChain:
        '0x0000000000000000000000000000000000000000' as const,
      BondManager: '0x0000000000000000000000000000000000000000' as const,
      OptimismPortal: '0xA581Ca3353DB73115C4625FFC7aDF5dB379434A8' as const,
      L2OutputOracle: '0x3A234299a14De50027eA65dCdf1c0DaC729e04A6' as const,
      OptimismPortal2: '0x0000000000000000000000000000000000000000' as const,
      DisputeGameFactory: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.BASE_GOERLI]: {
    l1: {
      AddressManager: '0x4Cf6b56b14c6CFcB72A75611080514F94624c54e' as const,
      L1CrossDomainMessenger:
        '0x8e5693140eA606bcEB98761d9beB1BC87383706D' as const,
      L1StandardBridge: '0xfA6D8Ee5BE770F84FC001D098C4bD604Fe01284a' as const,
      StateCommitmentChain:
        '0x0000000000000000000000000000000000000000' as const,
      CanonicalTransactionChain:
        '0x0000000000000000000000000000000000000000' as const,
      BondManager: '0x0000000000000000000000000000000000000000' as const,
      OptimismPortal: '0xe93c8cD0D409341205A592f8c4Ac1A5fe5585cfA' as const,
      L2OutputOracle: '0x2A35891ff30313CcFa6CE88dcf3858bb075A2298' as const,
      OptimismPortal2: '0x0000000000000000000000000000000000000000' as const,
      DisputeGameFactory: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.BASE_SEPOLIA]: {
    l1: {
      AddressManager: '0x709c2B8ef4A9feFc629A8a2C1AF424Dc5BD6ad1B' as const,
      L1CrossDomainMessenger:
        '0xC34855F4De64F1840e5686e64278da901e261f20' as const,
      L1StandardBridge: '0xfd0Bf71F60660E2f608ed56e1659C450eB113120' as const,
      StateCommitmentChain:
        '0x0000000000000000000000000000000000000000' as const,
      CanonicalTransactionChain:
        '0x0000000000000000000000000000000000000000' as const,
      BondManager: '0x0000000000000000000000000000000000000000' as const,
      OptimismPortal: '0x49f53e41452C74589E85cA1677426Ba426459e85' as const,
      L2OutputOracle: '0x84457ca9D0163FbC4bbfe4Dfbb20ba46e48DF254' as const,
      OptimismPortal2: '0x0000000000000000000000000000000000000000' as const,
      DisputeGameFactory: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.BASE_MAINNET]: {
    l1: {
      AddressManager: '0x8EfB6B5c4767B09Dc9AA6Af4eAA89F749522BaE2' as const,
      L1CrossDomainMessenger:
        '0x866E82a600A1414e583f7F13623F1aC5d58b0Afa' as const,
      L1StandardBridge: '0x3154Cf16ccdb4C6d922629664174b904d80F2C35' as const,
      StateCommitmentChain:
        '0x0000000000000000000000000000000000000000' as const,
      CanonicalTransactionChain:
        '0x0000000000000000000000000000000000000000' as const,
      BondManager: '0x0000000000000000000000000000000000000000' as const,
      OptimismPortal: '0x49048044D57e1C92A77f79988d21Fa8fAF74E97e' as const,
      L2OutputOracle: '0x56315b90c40730925ec5485cf004d835058518A0' as const,
      OptimismPortal2: '0x0000000000000000000000000000000000000000' as const,
      DisputeGameFactory: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  // Zora Goerli
  [L2ChainID.ZORA_GOERLI]: {
    l1: {
      AddressManager: '0x54f4676203dEDA6C08E0D40557A119c602bFA246' as const,
      L1CrossDomainMessenger:
        '0xD87342e16352D33170557A7dA1e5fB966a60FafC' as const,
      L1StandardBridge: '0x7CC09AC2452D6555d5e0C213Ab9E2d44eFbFc956' as const,
      StateCommitmentChain:
        '0x0000000000000000000000000000000000000000' as const,
      CanonicalTransactionChain:
        '0x0000000000000000000000000000000000000000' as const,
      BondManager: '0x0000000000000000000000000000000000000000' as const,
      OptimismPortal: '0xDb9F51790365e7dc196e7D072728df39Be958ACe' as const,
      L2OutputOracle: '0xdD292C9eEd00f6A32Ff5245d0BCd7f2a15f24e00' as const,
      OptimismPortal2: '0x0000000000000000000000000000000000000000' as const,
      DisputeGameFactory: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.ZORA_MAINNET]: {
    l1: {
      AddressManager: '0xEF8115F2733fb2033a7c756402Fc1deaa56550Ef' as const,
      L1CrossDomainMessenger:
        '0xdC40a14d9abd6F410226f1E6de71aE03441ca506' as const,
      L1StandardBridge: '0x3e2Ea9B92B7E48A52296fD261dc26fd995284631' as const,
      StateCommitmentChain:
        '0x0000000000000000000000000000000000000000' as const,
      CanonicalTransactionChain:
        '0x0000000000000000000000000000000000000000' as const,
      BondManager: '0x0000000000000000000000000000000000000000' as const,
      OptimismPortal: '0x1a0ad011913A150f69f6A19DF447A0CfD9551054' as const,
      L2OutputOracle: '0x9E6204F750cD866b299594e2aC9eA824E2e5f95c' as const,
      OptimismPortal2: '0x0000000000000000000000000000000000000000' as const,
      DisputeGameFactory: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
}

/**
 * Mapping of L1 chain IDs to the list of custom bridge addresses for each chain.
 */
export const BRIDGE_ADAPTER_DATA: {
  [ChainID in L2ChainID]?: BridgeAdapterData
} = {
  [L2ChainID.OPTIMISM]: {
    wstETH: {
      Adapter: DAIBridgeAdapter,
      l1Bridge: '0x76943C0D61395d8F2edF9060e1533529cAe05dE6' as const,
      l2Bridge: '0x8E01013243a96601a86eb3153F0d9Fa4fbFb6957' as const,
    },
    BitBTC: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: '0xaBA2c5F108F7E820C049D5Af70B16ac266c8f128' as const,
      l2Bridge: '0x158F513096923fF2d3aab2BcF4478536de6725e2' as const,
    },
    DAI: {
      Adapter: DAIBridgeAdapter,
      l1Bridge: '0x10E6593CDda8c58a1d0f14C5164B376352a55f2F' as const,
      l2Bridge: '0x467194771dAe2967Aef3ECbEDD3Bf9a310C76C65' as const,
    },
    ECO: {
      Adapter: ECOBridgeAdapter,
      l1Bridge: '0xAa029BbdC947F5205fBa0F3C11b592420B58f824' as const,
      l2Bridge: '0xAa029BbdC947F5205fBa0F3C11b592420B58f824' as const,
    },
  },
  [L2ChainID.OPTIMISM_GOERLI]: {
    DAI: {
      Adapter: DAIBridgeAdapter,
      l1Bridge: '0x05a388Db09C2D44ec0b00Ee188cD42365c42Df23' as const,
      l2Bridge: '0x467194771dAe2967Aef3ECbEDD3Bf9a310C76C65' as const,
    },
    ECO: {
      Adapter: ECOBridgeAdapter,
      l1Bridge: '0x9A4464D6bFE006715382D39D183AAf66c952a3e0' as const,
      l2Bridge: '0x6aA809bAeA2e4C057b3994127cB165119c6fc3B2' as const,
    },
  },
}
