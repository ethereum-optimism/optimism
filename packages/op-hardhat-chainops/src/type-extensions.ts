import 'hardhat/types/runtime'
import { Contract, Transaction, Signer, providers } from 'ethers'
import { Log } from '@ethersproject/abstract-provider'
import { OpNodeProvider } from '@eth-optimism/core-utils'

import { ParsedLog, ParsedTransaction, BuildOptimismOptions } from './plugin'

export interface OptimismConfig {
  [network: string]: OptimismNetworkConfig
}

export interface OptimismNetworkConfig {
  l2Url?: string
  opNodeUrl?: string
}

declare module 'hardhat/types/config' {
  interface HardhatUserConfig {
    optimism?: OptimismConfig
  }

  interface HardhatConfig {
    optimism?: OptimismConfig
  }
}

declare module 'hardhat/types/runtime' {
  interface HardhatRuntimeEnvironment {
    optimism: {
      init: (opts?: BuildOptimismOptions) => Promise<void>
      contracts: {
        [key: string]: Contract
      }
      parseLog: (log: Log) => ParsedLog
      parseTransaction: (tx: Transaction) => ParsedTransaction
      opNodeProvider: OpNodeProvider
      l1Signer: Signer
      l2Signer: Signer
      l2Provider: providers.StaticJsonRpcProvider
    }
  }
}
