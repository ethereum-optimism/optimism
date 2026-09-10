import { defineChain } from 'viem'
import { sepolia } from 'viem/chains'
import { chainConfig } from 'viem/op-stack'

import { addressesToViemContractConstant } from '@/addressSet.js'
import {
  interopJntV3_0Addresses,
  interopJntV3_1Addresses,
} from '@/chains/interopJntV3Addresses.js'
import type { Network } from '@/chains/types.js'

const sourceId = sepolia.id

/**
 * L2 chain A definition for interop-jnt-v3-0
 * @category interop-jnt-v3
 */
export const interopJntV3_0 = defineChain({
  ...chainConfig,
  id: 420120137,
  name: 'Interop JNT V3 0',
  rpcUrls: {
    default: {
      http: ['https://interop-jnt-v3-0.optimism.io'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Interop JNT V3 0 Block Explorer',
      url: '',
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(interopJntV3_0Addresses, sourceId),
  },
})

/**
 * L2 chain B definition for interop-jnt-v3-1
 * @category interop-jnt-v3
 */
export const interopJntV3_1 = defineChain({
  ...chainConfig,
  id: 420120138,
  name: 'Interop JNT V3 1',
  rpcUrls: {
    default: {
      http: ['https://interop-jnt-v3-1.optimism.io'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Interop JNT V3 1 Block Explorer',
      url: '',
    },
  },
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  sourceId,
  testnet: true,
  contracts: {
    ...chainConfig.contracts,
    ...addressesToViemContractConstant(interopJntV3_1Addresses, sourceId),
  },
})

export const interopJntV3Chains = [interopJntV3_0, interopJntV3_1]

export const interopJntV3Network: Network = {
  name: 'interop-jnt-v3',
  sourceChain: sepolia,
  chains: interopJntV3Chains,
}
