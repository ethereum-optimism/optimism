import { Chain } from '../interfaces'

export const DEPOSIT_CONFIRMATION_BLOCKS = {
  [Chain.MAINNET]: 50 as const,
  [Chain.GOERLI]: 12 as const,
  [Chain.KOVAN]: 12 as const,
  // 2 just for testing purposes
  [Chain.HARDHAT_LOCAL]: 2 as const,
}

export const CHAIN_BLOCK_TIMES = {
  [Chain.MAINNET]: 13 as const,
  [Chain.GOERLI]: 15 as const,
  [Chain.KOVAN]: 4 as const,
  [Chain.HARDHAT_LOCAL]: 1 as const,
}
