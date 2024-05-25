import {
  createTestClient,
  createPublicClient,
  createWalletClient,
  http,
} from 'viem'
import { goerli, optimismGoerli } from 'viem/chains'

// we should instead use .env to determine chain so we can support alternate l1/l2 pairs
const L1_CHAIN = goerli
const L2_CHAIN = optimismGoerli
const L1_RPC_URL = 'http://localhost:8545'
const L2_RPC_URL = 'http://localhost:9545'

const l1TestClient = createTestClient({
  mode: 'anvil',
  chain: L1_CHAIN,
  transport: http(L1_RPC_URL),
})

const l2TestClient = createTestClient({
  mode: 'anvil',
  chain: L2_CHAIN,
  transport: http(L2_RPC_URL),
})

const l1PublicClient = createPublicClient({
  chain: L1_CHAIN,
  transport: http(L1_RPC_URL),
})

const l2PublicClient = createPublicClient({
  chain: L2_CHAIN,
  transport: http(L2_RPC_URL),
})

const l1WalletClient = createWalletClient({
  chain: L1_CHAIN,
  transport: http(L1_RPC_URL),
})

const l2WalletClient = createWalletClient({
  chain: L2_CHAIN,
  transport: http(L2_RPC_URL),
})

export {
  l1TestClient,
  l2TestClient,
  l1PublicClient,
  l2PublicClient,
  l1WalletClient,
  l2WalletClient,
}
