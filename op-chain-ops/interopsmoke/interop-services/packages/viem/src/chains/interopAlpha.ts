import { defineChain } from 'viem'
import { sepolia } from 'viem/chains'
import { chainConfig } from 'viem/op-stack'

import { addressesToViemContractConstant } from '@/addressSet.js'
import {
  interopAlpha0Addresses,
  interopAlpha1Addresses,
} from '@/chains/interopAlphaAddresses.js'
import type { Network } from '@/chains/types.js'

const sourceId = sepolia.id

const interopAlpha0Blockscout = {
  name: 'Interop Alpha 0 Blockscout',
  url: 'https://optimism-interop-alpha-0.blockscout.com/',
}

/**
 * L2 chain A definition for interop-alpha-0
 * @category interop-alpha
 */
export const interopAlpha0 = defineChain({
  ...chainConfig,
  id: 420120000,
  name: 'Interop Alpha 0',
  rpcUrls: {
    default: {
      http: ['https://interop-alpha-0.optimism.io'],
    },
  },
  blockExplorers: {
    default: interopAlpha0Blockscout,
    blockscout: interopAlpha0Blockscout,
    routescan: {
      name: 'Interop Alpha 0 Routescan',
      url: 'https://420120000.testnet.routescan.io/',
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(interopAlpha0Addresses, sourceId),
  },
})

const interopAlpha1Blockscout = {
  name: 'Interop Alpha 1 Blockscout',
  url: 'https://optimism-interop-alpha-1.blockscout.com/',
}

/**
 * L2 chain A definition for interop-alpha-1
 * @category interop-alpha
 */
export const interopAlpha1 = defineChain({
  ...chainConfig,
  id: 420120001,
  name: 'Interop Alpha 1',
  rpcUrls: {
    default: {
      http: ['https://interop-alpha-1.optimism.io'],
    },
  },
  blockExplorers: {
    default: interopAlpha1Blockscout,
    blockscout: interopAlpha1Blockscout,
    routescan: {
      name: 'Interop Alpha 1 Routescan',
      url: 'https://420120001.testnet.routescan.io/',
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(interopAlpha1Addresses, sourceId),
  },
})

export const interopAlphaChains = [interopAlpha0, interopAlpha1]

export const interopAlphaNetwork: Network = {
  name: 'interop-alpha',
  sourceChain: sepolia,
  chains: interopAlphaChains,
}
