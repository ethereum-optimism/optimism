import type { Account, Chain } from 'viem'
import { createWalletClient } from 'viem'
import { createConnector, http } from 'wagmi'

const createProvider = (chain: Chain, account: Account) => {
  const accountClient = createWalletClient({
    account: account,
    chain: chain,
    transport: http(),
  })

  const request = async ({
    method,
    params,
  }: Parameters<typeof accountClient.request>) => {
    return accountClient.request({ method, params })
  }

  return { request }
}

// A connector that allows you to connect to a specific account on every chain
export const devAccount = (account: Account) => {
  return createConnector((config) => {
    let currentChain = config.chains[0]

    const providerByChainId = new Map<
      number,
      ReturnType<typeof createProvider>
    >()
    config.chains.forEach((chain) => {
      providerByChainId.set(chain.id, createProvider(chain, account))
    })

    const chainById = new Map<number, Chain>()
    config.chains.forEach((chain) => {
      chainById.set(chain.id, chain)
    })

    return {
      // icon
      id: 'devAccount',
      name: `Dev Account (${account.address.slice(
        0,
        4,
      )}...${account.address.slice(-4)})`,
      type: 'devAccount' as const,
      connect: async (params) => {
        currentChain = params?.chainId
          ? chainById.get(params.chainId) || config.chains[0]
          : config.chains[0]
        return {
          accounts: [account.address],
          chainId: currentChain.id,
        }
      },
      disconnect: async () => {
        return
      },
      getAccounts: async () => {
        return [account.address]
      },
      getChainId: async () => {
        return currentChain.id
      },

      getProvider: async (params) => {
        if (params?.chainId) {
          return providerByChainId.get(params.chainId)
        }
        return providerByChainId.get(currentChain.id)
      },
      isAuthorized: async () => {
        return true
      },
      switchChain: async ({ chainId }) => {
        const newChain = chainById.get(chainId)
        if (!newChain) {
          throw new Error(`Chain with id ${chainId} not found`)
        }
        currentChain = newChain

        config.emitter.emit('change', { chainId })
        return currentChain
      },
      onAccountsChanged: async () => {
        return
      },
      onChainChanged: async () => {
        return
      },
      onConnect: async () => {
        return
      },
      onDisconnect: async () => {
        return
      },
      onMessage: async () => {
        return
      },
    }
  })
}
