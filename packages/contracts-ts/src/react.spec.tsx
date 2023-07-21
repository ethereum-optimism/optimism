import matchers from '@testing-library/jest-dom/matchers'
import { cleanup, waitFor } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'
import { afterEach, expect, test } from 'vitest'
import { useMintManagerOwner } from './react'
import { configureChains, createConfig, WagmiConfig } from 'wagmi'
import * as React from 'react'
import { optimism } from 'viem/chains'
import { jsonRpcProvider } from 'wagmi/providers/jsonRpc'

expect.extend(matchers)

afterEach(() => {
  cleanup()
})

const { publicClient } = configureChains(
  [optimism],
  [
    jsonRpcProvider({
      rpc: () => ({
        http:
          import.meta.env.VITE_RPC_URL_L2_MAINNET ??
          'https://mainnet.optimism.io',
      }),
    }),
  ]
)

const config = createConfig({
  publicClient: ({ chainId }) => publicClient({ chainId }),
})

const blockNumber = BigInt(106806163)

test('react hooks should work', async () => {
  const hook = renderHook(
    () => useMintManagerOwner({ chainId: 10, blockNumber }),
    {
      wrapper: ({ children }) => (
        <WagmiConfig config={config}>{children}</WagmiConfig>
      ),
    }
  )
  await waitFor(() => {
    hook.rerender()
    if (hook.result.current.error) throw hook.result.current.error
    expect(hook.result.current?.data).toBeDefined()
  })
  const normalizedResult = {
    ...hook.result.current,
    internal: {
      ...hook.result.current.internal,
      dataUpdatedAt: 'SNAPSHOT_TEST_REMOVED!!!',
    },
  }
  expect(normalizedResult).toMatchInlineSnapshot(`
    {
      "data": "0x2A82Ae142b2e62Cb7D10b55E323ACB1Cab663a26",
      "error": null,
      "fetchStatus": "idle",
      "internal": {
        "dataUpdatedAt": "SNAPSHOT_TEST_REMOVED!!!",
        "errorUpdatedAt": 0,
        "failureCount": 0,
        "isFetchedAfterMount": true,
        "isLoadingError": false,
        "isPaused": false,
        "isPlaceholderData": false,
        "isPreviousData": false,
        "isRefetchError": false,
        "isStale": true,
        "remove": [Function],
      },
      "isError": false,
      "isFetched": true,
      "isFetchedAfterMount": true,
      "isFetching": false,
      "isIdle": false,
      "isLoading": false,
      "isRefetching": false,
      "isSuccess": true,
      "refetch": [Function],
      "status": "success",
    }
  `)
})
