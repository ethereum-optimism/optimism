import {
  createPublicClient,
  createTestClient,
  createWalletClient,
  http,
} from 'viem'
import { privateKeyToAccount } from 'viem/accounts'

import { supersimL2A, supersimL2B } from '@/chains/supersim.js'
import { publicActionsL2 } from '@/decorators/publicL2.js'
import { walletActionsL2 } from '@/decorators/walletL2.js'

// anvil account: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
export const testAccount = privateKeyToAccount(
  '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
)

/** Chain A **/

export const publicClientA = createPublicClient({
  chain: supersimL2A,
  transport: http(supersimL2A.rpcUrls.default.http[0]),
}).extend(publicActionsL2())

export const walletClientA = createWalletClient({
  account: testAccount,
  chain: supersimL2A,
  transport: http(supersimL2A.rpcUrls.default.http[0]),
}).extend(walletActionsL2())

export const testClientA = createTestClient({
  chain: supersimL2A,
  mode: 'anvil',
  transport: http(supersimL2A.rpcUrls.default.http[0]),
})

/** Chain B **/

export const publicClientB = createPublicClient({
  chain: supersimL2B,
  transport: http(supersimL2B.rpcUrls.default.http[0]),
}).extend(publicActionsL2())

export const walletClientB = createWalletClient({
  account: testAccount,
  chain: supersimL2B,
  transport: http(supersimL2B.rpcUrls.default.http[0]),
}).extend(walletActionsL2())

export const testClientB = createTestClient({
  chain: supersimL2A,
  mode: 'anvil',
  transport: http(supersimL2B.rpcUrls.default.http[0]),
})
