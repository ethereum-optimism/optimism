import ethers from 'ethers'
import { z } from 'zod'

/**
 * @deprecated
 */
const E2E_RPC_URL_L1 = z
  .string()
  .url()
  .default('http://localhost:8545')
  .describe('L1 ethereum rpc Url')
  .parse(import.meta.env.VITE_E2E_RPC_URL_L1)
/**
 * @deprecated
 */
const E2E_RPC_URL_L2 = z
  .string()
  .url()
  .default('http://localhost:9545')
  .describe('L2 ethereum rpc Url')
  .parse(import.meta.env.VITE_E2E_RPC_URL_L2)

const jsonRpcHeaders = { 'User-Agent': 'eth-optimism/@gateway/backend' }
/**
 * @deprecated
 */
export const l1Provider = new ethers.providers.JsonRpcProvider({
  url: E2E_RPC_URL_L1,
  headers: jsonRpcHeaders,
})
/**
 * @deprecated
 */
export const l2Provider = new ethers.providers.JsonRpcProvider({
  url: E2E_RPC_URL_L2,
  headers: jsonRpcHeaders,
})

export const E2E_RPC_URL_SEPOLIA = z
  .string()
  .url()
  .default('http://localhost:8545')
  .describe('SEPOLIA ethereum rpc Url')
  .parse(import.meta.env.VITE_E2E_RPC_URL_SEPOLIA)
export const E2E_RPC_URL_OP_SEPOLIA = z
  .string()
  .url()
  .default('http://localhost:9545')
  .describe('OP_SEPOLIA ethereum rpc Url')
  .parse(import.meta.env.VITE_E2E_RPC_URL_OP_SEPOLIA)
export const sepoliaProvider = new ethers.providers.JsonRpcProvider({
  url: E2E_RPC_URL_SEPOLIA,
  headers: jsonRpcHeaders,
})
export const opSepoliaProvider = new ethers.providers.JsonRpcProvider({
  url: E2E_RPC_URL_OP_SEPOLIA,
  headers: jsonRpcHeaders,
})
