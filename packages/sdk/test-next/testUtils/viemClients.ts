import {
  createTestClient,
  createPublicClient,
  createWalletClient,
  http,
} from 'viem'
import { goerli, optimismGoerli, optimismSepolia, sepolia } from 'viem/chains'

import { E2E_RPC_URL_OP_SEPOLIA, E2E_RPC_URL_SEPOLIA } from './ethersProviders'

/**
 * @deprecated
 */
const L1_CHAIN = goerli
/**
 * @deprecated
 */
const L2_CHAIN = optimismGoerli
/**
 * @deprecated
 */
const L1_RPC_URL = 'http://localhost:8545'
/**
 * @deprecated
 */
const L2_RPC_URL = 'http://localhost:9545'

/**
 * @deprecated
 */
export const l1TestClient = createTestClient({
  mode: 'anvil',
  chain: L1_CHAIN,
  transport: http(L1_RPC_URL),
})

/**
 * @deprecated
 */
export const l2TestClient = createTestClient({
  mode: 'anvil',
  chain: L2_CHAIN,
  transport: http(L2_RPC_URL),
})

/**
 * @deprecated
 */
export const l1PublicClient = createPublicClient({
  chain: L1_CHAIN,
  transport: http(L1_RPC_URL),
})

/**
 * @deprecated
 */
export const l2PublicClient = createPublicClient({
  chain: L2_CHAIN,
  transport: http(L2_RPC_URL),
})

/**
 * @deprecated
 */
export const l1WalletClient = createWalletClient({
  chain: L1_CHAIN,
  transport: http(L1_RPC_URL),
})

/**
 * @deprecated
 */
export const l2WalletClient = createWalletClient({
  chain: L2_CHAIN,
  transport: http(L2_RPC_URL),
})

const SEPOLIA_CHAIN = sepolia
const OP_SEPOLIA_CHAIN = optimismSepolia

export const sepoliaTestClient = createTestClient({
  mode: 'anvil',
  chain: SEPOLIA_CHAIN,
  transport: http(E2E_RPC_URL_SEPOLIA),
})

export const opSepoliaTestClient = createTestClient({
  mode: 'anvil',
  chain: OP_SEPOLIA_CHAIN,
  transport: http(E2E_RPC_URL_OP_SEPOLIA),
})

export const sepoliaPublicClient = createPublicClient({
  chain: SEPOLIA_CHAIN,
  transport: http(E2E_RPC_URL_SEPOLIA),
})

export const opSepoliaPublicClient = createPublicClient({
  chain: OP_SEPOLIA_CHAIN,
  transport: http(E2E_RPC_URL_OP_SEPOLIA),
})

export const sepoliaWalletClient = createWalletClient({
  chain: SEPOLIA_CHAIN,
  transport: http(E2E_RPC_URL_SEPOLIA),
})

export const opSepoliaWalletClient = createWalletClient({
  chain: OP_SEPOLIA_CHAIN,
  transport: http(E2E_RPC_URL_OP_SEPOLIA),
})
