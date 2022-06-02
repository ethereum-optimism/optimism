// Types explicitly related to dealing with Geth.

/**
 * Represents the Ethereum state, in the format that Geth expects it.
 */
export interface State {
  [address: string]: {
    nonce: number
    balance: string
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
  mergeForkBlock?: number
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
  nonce?: number
  timestamp?: number
  difficulty: string
  mixHash?: string
  coinbase?: string
  gasLimit: string
  extraData: string
  alloc: State
}
