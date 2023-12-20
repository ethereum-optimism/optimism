import addressesJson from '@eth-optimism/superchain-registry/superchain/extra/addresses/addresses.json'
import { writeFileSync } from 'fs'
import * as viemChains from 'viem/chains/opStack/chains.js'
import { camelCase } from 'change-case'
import { getL2Predeploys } from './getL2Predeploys.js'
import YAML from 'yaml'
import { OP_GENESIS_BLOCK } from './OP_GENESIS_BLOCK.js'
import { Chain } from 'viem'

writeFileSync('./src/chains.ts', generateChainsFile())

/**
 * Reads chain information from @eth-optimism/superchain-registry and generates a file shaped like a viem chain
 */
function generateChainsFile() {
  /**
   * Loops through every superchain chain and generates a viem chain to be used in typescript
   * Currently it uses addresses.json but a nice cleanup here would be to use the yaml files directly and then
   * we can remove the addresses.json script in favor of this script
   */
  const chains: Record<number, Chain> = {}
  Object.entries(addressesJson).forEach(([chainIdStr, contracts]) => {
    const chainId = parseInt(chainIdStr)
    const genesisBlock = chainId === 10 ? OP_GENESIS_BLOCK : 0
    const viemChain = Object.values(viemChains).find(chain => chain.id === chainId)
    const { ProxyAdmin, OptimismPortalProxy, AddressManager, L1ERC721BridgeProxy, L2OutputOracleProxy, L1StandardBridgeProxy, L1CrossDomainMessengerProxy, OptimismMintableERC20FactoryProxy } = contracts
    // TODO add the `blockCreated` property to all of these
    const viemContracts =
      {
        ...getL2Predeploys(genesisBlock),
        portal: {
          ...(viemChain as any)?.contracts?.portal,
          address: OptimismPortalProxy as `0x${string}`,

        },
        addressManager: {
          address: AddressManager as `0x${string}`,
        },
        proxyAdmin: {
          address: ProxyAdmin as `0x${string}`,
        },
        l1ERC721Bridge: {
          address: L1ERC721BridgeProxy as `0x${string}`,
        },
        l2OutputOracle: {
          address: L2OutputOracleProxy as `0x${string}`,
        },
        l1StandardBridge: {
          address: L1StandardBridgeProxy as `0x${string}`,
        },
        l1CrossDomainMessenger: {
          address: L1CrossDomainMessengerProxy as `0x${string}`,
        },
        l2ERC20Factory: {
          address: OptimismMintableERC20FactoryProxy as `0x${string}`,
        },
      } as const
    if (!viemChain) {
      console.warn(`no viem chain found for superchain chain ${chainId}! Please notify this chain partner to do a pr to viem`)
      chains[chainId] = {
        id: chainId,
        // Update viem chains so this name is correct
        name: `opstack${chainId}`,
        nativeCurrency: viemChains.optimism.nativeCurrency,
        // update viem so these are filled in
        rpcUrls: {
          default: {
            http: [],
          },
          public: {
            http: [],
          },
        },
        contracts: {
          ...viemContracts
        },
        // TODO we are assuming an unknown chain has a sourceId of 1 (mainnet)
        sourceId: 1
      }
    } else {
      chains[chainId] = {
        ...viemChain,
        contracts: {
          ...viemChain.contracts,
          ...viemContracts,
        }
      }
    }
  })

  const file: string[] = []

  Object.values(chains).forEach(chain => {
    file.push(`export const ${(camelCase(chain.name))} = ${JSON.stringify(chain, null, 2)}`)
  })

  // EOL
  file.push('\n')

  return file.join('\n')
}

