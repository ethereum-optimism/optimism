export interface ChainState {
  [address: string]: {
    balance: string
    nonce: number
    root: string
    codeHash: string
    code?: string
    storage?: {
      [key: string]: string
    }
  }
}

export interface StateDump {
  root: string
  accounts: ChainState
}
