import { defineChain, multicall3Abi } from 'viem'
import { chainConfig } from 'viem/op-stack'

import { addressesToViemContractConstant } from '@/addressSet.js'
import {
  supersimL2AAddresses,
  supersimL2BAddresses,
  supersimL2CAddresses,
  supersimL2DAddresses,
  supersimL2EAddresses,
} from '@/chains/supersimAddresses.js'
import type { Network } from '@/chains/types.js'

/**
 * L1 chain definition for supersim in non-forked mode
 * @category Supersim
 */
export const supersimL1 = defineChain({
  ...chainConfig,
  id: 900,
  name: 'Supersim L1',
  rpcUrls: {
    default: {
      http: ['http://127.0.0.1:8545'],
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  testnet: true,
  contracts: {
    multicall3: {
      address: '0xca11bde05977b3631167028862be2a173976ca11',
      abi: multicall3Abi,
    },
  },
})

const sourceId = supersimL1.id

/**
 * L2 chain A definition for supersim in non-forked mode
 * @category Supersim
 */
export const supersimL2A = defineChain({
  ...chainConfig,
  id: 901,
  name: 'Supersim L2 A',
  rpcUrls: {
    default: {
      http: ['http://127.0.0.1:9545'],
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(supersimL2AAddresses, sourceId),
  },
})

/**
 * L2 chain B definition for supersim in non-forked mode
 * @category Supersim
 */
export const supersimL2B = defineChain({
  ...chainConfig,
  id: 902,
  name: 'Supersim L2 B',
  rpcUrls: {
    default: {
      http: ['http://127.0.0.1:9546'],
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(supersimL2BAddresses, sourceId),
  },
})

/**
 * L2 chain C definition for supersim in non-forked mode
 * @category Supersim
 */
export const supersimL2C = defineChain({
  ...chainConfig,
  id: 903,
  name: 'Supersim L2 C',
  rpcUrls: {
    default: {
      http: ['http://127.0.0.1:9547'],
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(supersimL2CAddresses, sourceId),
  },
})

/**
 * L2 chain D definition for supersim in non-forked mode
 * @category Supersim
 */
export const supersimL2D = defineChain({
  ...chainConfig,
  id: 904,
  name: 'Supersim L2 D',
  rpcUrls: {
    default: {
      http: ['http://127.0.0.1:9548'],
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(supersimL2DAddresses, sourceId),
  },
})

/**
 * L2 chain E definition for supersim in non-forked mode
 * @category Supersim
 */
export const supersimL2E = defineChain({
  ...chainConfig,
  id: 905,
  name: 'Supersim L2 E',
  rpcUrls: {
    default: {
      http: ['http://127.0.0.1:9549'],
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(supersimL2EAddresses, sourceId),
  },
})

export const supersimChains = [
  supersimL2A,
  supersimL2B,
  supersimL2C,
  supersimL2D,
  supersimL2E,
]

export const supersimNetwork: Network = {
  name: 'supersim',
  sourceChain: supersimL1,
  chains: supersimChains,
}
