import type { NetworkName } from '@eth-optimism/viem/chains'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Config = {
  networkName: NetworkName
  setNetworkName: (networkName: NetworkName) => void

  rpcOverrideByChainId: Record<number, string>
  setRpcOverrideByChainId: (chainId: number, rpcUrl: string) => void
}

export const useConfig = create<Config>()(
  persist(
    (set, get) => ({
      networkName: 'sepolia', // Default to Sepolia
      setNetworkName: (networkName: NetworkName) => {
        set({ networkName })
      },

      rpcOverrideByChainId: {},
      setRpcOverrideByChainId: (chainId: number, rpcUrl: string) => {
        set({
          rpcOverrideByChainId: {
            ...get().rpcOverrideByChainId,
            [chainId]: rpcUrl,
          },
        })
      },
    }),
    {
      name: 'superchain-tools-storage',
    },
  ),
)
