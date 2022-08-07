// Types explicitly related to dealing with Geth.

/**
 * Represents the Ethereum state, in the format that Geth expects it.
 */
export interface State {
  [address: string]: {
    nonce?: string
    balance?: string
    codeHash?: string
    root?: string
    code?: string
    storage?: {
      [key: string]: string
    }
    secretKey?: string
  }
}

/**
 * Represents Geth's ChainConfig
 */
export interface ChainConfig {
  chainId: number
  homesteadBlock: number
  eip150Block: number
  eip150Hash?: string
  eip155Block: number
  eip158Block: number
  byzantiumBlock: number
  constantinopleBlock: number
  petersburgBlock: number
  istanbulBlock: number
  muirGlacierBlock: number
  berlinBlock: number
  londonBlock?: number
  arrowGlacierBlock?: number
  grayGlacierBlock?: number
  mergeNetsplitBlock?: number
  terminalTotalDifficulty?: number
  clique?: {
    period: number
    epoch: number
  }
  ethash?: {}
}

/**
 * Represents Geth's genesis file format.
 */
export interface Genesis {
  config: ChainConfig
  nonce?: string
  timestamp?: string
  difficulty: string
  mixHash?: string
  coinbase?: string
  number?: string
  gasLimit: string
  gasUsed?: string
  parentHash?: string
  extraData: string
  baseFeePerGas?: string
  alloc: State
}

/**
 * Represents the chain config for an Optimism chain
 */
export interface OptimismChainConfig extends ChainConfig {
  optimism: {
    baseFeeRecipient: string
    l1FeeRecipient: string
  }
}

/**
 * Represents the Genesis file format for an Optimism chain
 */
export interface OptimismGenesis extends Genesis {
  config: OptimismChainConfig
}
