import { predeploys } from '@eth-optimism/contracts'

import {
  L1ChainID,
  L2ChainID,
  OEContractsLike,
  OEL2ContractsLike,
  BridgeAdapterData,
} from '../interfaces'
import { StandardBridgeAdapter, DAIBridgeAdapter } from '../adapters'

export const DEPOSIT_CONFIRMATION_BLOCKS: {
  [ChainID in L2ChainID]: number
} = {
  [L2ChainID.OPTIMISM]: 50 as const,
  [L2ChainID.OPTIMISM_GOERLI]: 12 as const,
  [L2ChainID.OPTIMISM_KOVAN]: 12 as const,
  [L2ChainID.OPTIMISM_HARDHAT_LOCAL]: 2 as const,
  [L2ChainID.OPTIMISM_HARDHAT_DEVNET]: 2 as const,
  [L2ChainID.OPTIMISM_BEDROCK_LOCAL_DEVNET]: 2 as const,
}

export const CHAIN_BLOCK_TIMES: {
  [ChainID in L1ChainID]: number
} = {
  [L1ChainID.MAINNET]: 13 as const,
  [L1ChainID.GOERLI]: 15 as const,
  [L1ChainID.KOVAN]: 4 as const,
  [L1ChainID.HARDHAT_LOCAL]: 1 as const,
  [L1ChainID.BEDROCK_LOCAL_DEVNET]: 15 as const,
}

/**
 * Full list of default L2 contract addresses.
 * TODO(tynes): migrate to predeploys from contracts-bedrock
 */
export const DEFAULT_L2_CONTRACT_ADDRESSES: OEL2ContractsLike = {
  L2CrossDomainMessenger: predeploys.L2CrossDomainMessenger,
  L2ToL1MessagePasser: predeploys.OVM_L2ToL1MessagePasser,
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
 * Mapping of L1 chain IDs to the appropriate contract addresses for the OE deployments to the
 * given network. Simplifies the process of getting the correct contract addresses for a given
 * contract name.
 */
export const CONTRACT_ADDRESSES: {
  [ChainID in L2ChainID]: OEContractsLike
} = {
  [L2ChainID.OPTIMISM]: {
    l1: {
      AddressManager: '0xdE1FCfB0851916CA5101820A69b13a4E276bd81F' as const,
      L1CrossDomainMessenger:
        '0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1' as const,
      L1StandardBridge: '0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1' as const,
      StateCommitmentChain:
        '0xBe5dAb4A2e9cd0F27300dB4aB94BeE3A233AEB19' as const,
      CanonicalTransactionChain:
        '0x5E4e65926BA27467555EB562121fac00D24E9dD2' as const,
      BondManager: '0xcd626E1328b41fCF24737F137BcD4CE0c32bc8d1' as const,
      OptimismPortal: '0x0000000000000000000000000000000000000000' as const,
      L2OutputOracle: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.OPTIMISM_KOVAN]: {
    l1: {
      AddressManager: '0x100Dd3b414Df5BbA2B542864fF94aF8024aFdf3a' as const,
      L1CrossDomainMessenger:
        '0x4361d0F75A0186C05f971c566dC6bEa5957483fD' as const,
      L1StandardBridge: '0x22F24361D548e5FaAfb36d1437839f080363982B' as const,
      StateCommitmentChain:
        '0xD7754711773489F31A0602635f3F167826ce53C5' as const,
      CanonicalTransactionChain:
        '0xf7B88A133202d41Fe5E2Ab22e6309a1A4D50AF74' as const,
      BondManager: '0xc5a603d273E28185c18Ba4d26A0024B2d2F42740' as const,
      OptimismPortal: '0x0000000000000000000000000000000000000000' as const,
      L2OutputOracle: '0x0000000000000000000000000000000000000000' as const,
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.OPTIMISM_GOERLI]: {
    l1: {
      AddressManager: '0xfA5b622409E1782597952a4A78c1D34CF32fF5e2' as const,
      L1CrossDomainMessenger:
        '0x5086d1eEF304eb5284A0f6720f79403b4e9bE294' as const,
      L1StandardBridge: '0x636Af16bf2f682dD3109e60102b8E1A089FedAa8' as const,
      StateCommitmentChain:
        '0x9c945aC97Baf48cB784AbBB61399beB71aF7A378' as const,
      CanonicalTransactionChain:
        '0x607F755149cFEB3a14E1Dc3A4E2450Cde7dfb04D' as const,
      BondManager: '0xfC2ab6987C578218f99E85d61Dcf4814A26637Bd' as const,
      OptimismPortal: '0x0000000000000000000000000000000000000000' as const,
      L2OutputOracle: '0x0000000000000000000000000000000000000000' as const,
    },
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
      OptimismPortal: '0x0000000000000000000000000000000000000000' as const,
      L2OutputOracle: '0x0000000000000000000000000000000000000000' as const,
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
    },
    l2: DEFAULT_L2_CONTRACT_ADDRESSES,
  },
  [L2ChainID.OPTIMISM_BEDROCK_LOCAL_DEVNET]: {
    l1: {
      AddressManager: '0x5FbDB2315678afecb367f032d93F642f64180aa3' as const,
      L1CrossDomainMessenger:
        '0x0165878A594ca255338adfa4d48449f69242Eb8F' as const,
      L1StandardBridge: '0x8A791620dd6260079BF849Dc5567aDC3F2FdC318' as const,
      StateCommitmentChain:
        '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9' as const,
      CanonicalTransactionChain:
        '0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9' as const,
      BondManager: '0x5FC8d32690cc91D4c39d9d3abcBD16989F875707' as const,
      OptimismPortal: '0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9' as const,
      L2OutputOracle: '0x5FbDB2315678afecb367f032d93F642f64180aa3' as const,
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
  },
  [L2ChainID.OPTIMISM_KOVAN]: {
    wstETH: {
      Adapter: DAIBridgeAdapter,
      l1Bridge: '0x65321bf24210b81500230dCEce14Faa70a9f50a7' as const,
      l2Bridge: '0x2E34e7d705AfaC3C4665b6feF31Aa394A1c81c92' as const,
    },
    BitBTC: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: '0x0b651A42F32069d62d5ECf4f2a7e5Bd3E9438746' as const,
      l2Bridge: '0x0CFb46528a7002a7D8877a5F7a69b9AaF1A9058e' as const,
    },
    USX: {
      Adapter: StandardBridgeAdapter,
      l1Bridge: '0x40E862341b2416345F02c41Ac70df08525150dC7' as const,
      l2Bridge: '0xB4d37826b14Cd3CB7257A2A5094507d701fe715f' as const,
    },
    DAI: {
      Adapter: DAIBridgeAdapter,
      l1Bridge: '0xb415e822C4983ecD6B1c1596e8a5f976cf6CD9e3' as const,
      l2Bridge: '0x467194771dAe2967Aef3ECbEDD3Bf9a310C76C65' as const,
    },
  },
}
