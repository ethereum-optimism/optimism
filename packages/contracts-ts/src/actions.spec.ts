import { test, expect } from 'vitest'
import { getOptimismPortal } from './actions'
import { createConfig, configureChains } from 'wagmi'
import { jsonRpcProvider } from 'wagmi/providers/jsonRpc'
import { mainnet } from 'viem/chains'

const { publicClient } = configureChains(
  [mainnet],
  [
    jsonRpcProvider({
      rpc: () => ({
        http:
          import.meta.env.VITE_RPC_URL_L1_MAINNET ?? mainnet.rpcUrls.default[0],
      }),
    }),
  ]
)

createConfig({
  publicClient: ({ chainId }) => publicClient({ chainId }),
})

const blockNumber = BigInt(17681356)

test('should be able to create a wagmi contract and use it', async () => {
  const portal = getOptimismPortal({ chainId: 1 })
  expect(portal).toBeDefined()
  expect(await portal.read.version({ blockNumber })).toMatchInlineSnapshot(
    '"1.6.0"'
  )
})
