import type { Hex } from 'viem'

export const truncateHash = (address: Hex) => {
  return `${address.slice(0, 10)}...${address.slice(-6)}`
}
