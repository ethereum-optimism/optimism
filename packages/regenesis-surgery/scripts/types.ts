import { ethers } from 'ethers'

export interface SurgeryConfigs {
  stateDumpFilePath: string
  etherscanFilePath: string
  genesisFilePath: string
  outputFilePath: string
  l2NetworkName?: SupportedNetworks
  l2ProviderUrl: string
  ropstenProviderUrl: string
  ropstenPrivateKey: string
  ethProviderUrl: string
  stateDumpHeight: number
  startIndex: number
  endIndex: number
}

export interface Account {
  address: string
  nonce: number | string
  balance: string
  codeHash?: string
  root?: string
  code?: string
  storage?: {
    [key: string]: string
  }
}

export type StateDump = Account[]

export interface GethStateDump {
  [address: string]: {
    nonce: number
    balance: string
    codeHash: string
    root: string
    code?: string
    storage?: {
      [key: string]: string
    }
  }
}

export enum AccountType {
  ONEINCH_DEPLOYER,
  DELETE,
  EOA,
  PRECOMPILE,
  PREDEPLOY_NEW_NOT_ETH,
  PREDEPLOY_WIPE,
  PREDEPLOY_NO_WIPE,
  PREDEPLOY_ETH,
  PREDEPLOY_WETH,
  UNISWAP_V3_FACTORY,
  UNISWAP_V3_NFPM,
  UNISWAP_V3_MAINNET_MULTICALL,
  UNISWAP_V3_POOL,
  UNISWAP_V3_OTHER,
  UNVERIFIED,
  VERIFIED,
  ERC20,
}

export interface UniswapPoolData {
  oldAddress: string
  newAddress: string
  token0: string
  token1: string
  fee: ethers.BigNumber
}

export interface EtherscanContract {
  contractAddress: string
  code: string
  hash: string
  sourceCode: string
  creationCode: string
  contractFileName: string
  contractName: string
  compilerVersion: string
  optimizationUsed: string
  runs: string
  constructorArguments: string
  library: string
}

export type EtherscanDump = EtherscanContract[]

export type SupportedNetworks = 'mainnet' | 'kovan'

export interface SurgeryDataSources {
  configs: SurgeryConfigs
  dump: StateDump
  genesis: GenesisFile
  genesisDump: StateDump
  pools: UniswapPoolData[]
  poolHashCache: PoolHashCache
  etherscanDump: EtherscanContract[]
  ropstenProvider: ethers.providers.JsonRpcProvider
  ropstenWallet: ethers.Wallet
  l2Provider: ethers.providers.JsonRpcProvider
  ethProvider: ethers.providers.JsonRpcProvider
}

export interface GenesisFile {
  config: {
    chainId: number
    homesteadBlock: number
    eip150Block: number
    eip155Block: number
    eip158Block: number
    byzantiumBlock: number
    constantinopleBlock: number
    petersburgBlock: number
    istanbulBlock: number
    muirGlacierBlock: number
    clique: {
      period: number
      epoch: number
    }
  }
  difficulty: string
  gasLimit: string
  extraData: string
  alloc: GethStateDump
}

export interface ImmutableReference {
  start: number
  length: number
}

export interface ImmutableReferences {
  [key: string]: ImmutableReference[]
}

export interface PoolHashCache {
  [key: string]: {
    pool: UniswapPoolData
    index: number
  }
}
