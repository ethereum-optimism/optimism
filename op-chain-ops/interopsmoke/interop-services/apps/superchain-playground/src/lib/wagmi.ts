import '@rainbow-me/rainbowkit/styles.css'

import { networks } from '@eth-optimism/viem/chains'
import type { Chain } from 'viem'
import { privateKeyToAccount } from 'viem/accounts'
import { createConfig, http } from 'wagmi'
import { metaMask, walletConnect } from 'wagmi/connectors'

import { devAccount } from '@/connectors/devAccount'
import { envVars } from '@/envVars'

export const defaultDevAccount = privateKeyToAccount(
  '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
)

const ns = Object.values(networks)

const sourceChains = ns.map((n) => n.sourceChain)
const chains = ns.flatMap((n) => n.chains)
const allChains = [...sourceChains, ...chains] as const

const transports = Object.fromEntries(
  allChains.map((chain) => {
    return [chain.id, http(chain.rpcUrls.default.http[0])]
  }),
)

export const config = createConfig({
  chains: allChains as readonly [Chain, ...Chain[]],
  transports,
  connectors: [
    devAccount(defaultDevAccount),
    walletConnect({ projectId: envVars.VITE_WALLET_CONNECT_PROJECT_ID || '' }),
    metaMask(),
  ],
})
