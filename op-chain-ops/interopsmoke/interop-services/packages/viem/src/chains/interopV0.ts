import { defineChain } from 'viem'
import { sepolia } from 'viem/chains'
import { chainConfig } from 'viem/op-stack'

import { addressesToViemContractConstant } from '@/addressSet.js'
import {
  interopV0_0Addresses,
  interopV0_1Addresses,
} from '@/chains/interopV0Addresses.js'
import type { Network } from '@/chains/types.js'

const sourceId = sepolia.id

/**
 * L2 chain A definition for interop-v0-0
 * @category interop-v0
 */
export const interopV0_0 = defineChain({
  ...chainConfig,
  id: 420120046,
  name: 'Interop V0 0',
  rpcUrls: {
    default: {
      http: ['https://interop-v0-0.optimism.io'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Interop V0 0 Block Explorer',
      url: '',
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(interopV0_0Addresses, sourceId),
  },
})

/**
 * L2 chain B definition for interop-v0-1
 * @category interop-v0
 */
export const interopV0_1 = defineChain({
  ...chainConfig,
  id: 420120047,
  name: 'Interop V0 1',
  rpcUrls: {
    default: {
      http: ['https://interop-v0-1.optimism.io'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Interop V0 1 Block Explorer',
      url: '',
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(interopV0_1Addresses, sourceId),
  },
})

export const interopV0Chains = [interopV0_0, interopV0_1]

export const interopV0Network: Network = {
  name: 'interop-v0',
  sourceChain: sepolia,
  chains: interopV0Chains,
}
