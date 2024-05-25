import ethers from 'ethers'
import { z } from 'zod'

const E2E_RPC_URL_L1 = z
  .string()
  .url()
  .describe('L1 ethereum rpc Url')
  .parse(import.meta.env.VITE_E2E_RPC_URL_L1)
const E2E_RPC_URL_L2 = z
  .string()
  .url()
  .describe('L1 ethereum rpc Url')
  .parse(import.meta.env.VITE_E2E_RPC_URL_L2)

const jsonRpcHeaders = { 'User-Agent': 'eth-optimism/@gateway/backend' }
/**
 * Initialize the signer, prover, and cross chain messenger
 */
const l1Provider = new ethers.providers.JsonRpcProvider({
  url: E2E_RPC_URL_L1,
  headers: jsonRpcHeaders,
})
const l2Provider = new ethers.providers.JsonRpcProvider({
  url: E2E_RPC_URL_L2,
  headers: jsonRpcHeaders,
})

export { l1Provider, l2Provider }
