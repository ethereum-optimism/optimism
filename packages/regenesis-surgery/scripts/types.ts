import { ethers } from 'ethers'

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

export interface PoolData {
  oldAddress: string
  newAddress: string
  token0: string
  token1: string
  fee: ethers.BigNumber
}
