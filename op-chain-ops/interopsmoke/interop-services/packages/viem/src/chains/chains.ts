import type { Chain } from 'viem'

import { networks } from '@/chains/networks.js'

/**
 * Map of all viem chains by ID
 */
export const chainById = Object.values(networks)
  .flatMap((network) => network.chains)
  .reduce((acc, chain) => {
    acc[chain.id] = chain
    return acc
  }, {} as Record<number, Chain>)
