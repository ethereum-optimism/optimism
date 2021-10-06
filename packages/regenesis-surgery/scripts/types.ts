import { ethers } from 'ethers'

export interface Account {
  address: string
  nonce: number
  balance: string
  codeHash: string
  root: string
  code?: string
  storage?: {
    [key: string]: string
  }
}

export type StateDump = Account[]

export enum AccountType {
  EOA,
  PRECOMPILE,
  PREDEPLOY_DEAD,
  PREDEPLOY_WIPE,
  PREDEPLOY_NO_WIPE,
  PREDEPLOY_ETH,
  PREDEPLOY_WETH,
  UNISWAP_V3_FACTORY,
  UNISWAP_V3_NFPM,
  UNISWAP_V3_POOL,
  UNISWAP_V3_LIB,
  UNISWAP_V3_OTHER,
  UNVERIFIED,
  VERIFIED,
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

export interface SurgeryDataSources {
  dump: StateDump
  genesis: StateDump
  pools: UniswapPoolData[]
  etherscanDump: EtherscanContract[]
  l1TestnetProvider: ethers.providers.JsonRpcProvider
  l1TestnetWallet: ethers.Wallet
  l1MainnetProvider: ethers.providers.JsonRpcProvider
  l2Provider: ethers.providers.JsonRpcProvider
}
