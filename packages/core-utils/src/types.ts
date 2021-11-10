// Optimism PBC 2021

// Represents the ethereum state
export interface State {
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

// Represents a genesis file that geth can consume
export interface Genesis {
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
  alloc: State
}
