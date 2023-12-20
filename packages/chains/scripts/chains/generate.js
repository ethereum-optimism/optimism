import addressesJson from '@eth-optimism/superchain-registry/superchain/extra/addresses/addresses.json'
import { writeFileSync } from 'fs'
import * as viemChains from 'viem/op-stack'
import { camelCase } from 'change-case'
import { getL2Predeploys } from './getL2Predeploys.js'
import { OP_GENESIS_BLOCK } from './OP_GENESIS_BLOCK.js'
import { CHAINS_OUTPUT_PATH } from './paths.js'

writeFileSync(CHAINS_OUTPUT_PATH, generateChainsFile())

/**
 * Reads chain information from @eth-optimism/superchain-registry and generates a file shaped like a viem chain
 */
function generateChainsFile() {
  /**
   * @type {Record<number, import('viem').Chain>}
   * Loops through every superchain chain and generates a viem chain to be used in typescript
   * Currently it uses addresses.json but a nice cleanup here would be to use the yaml files directly and then
   * we can remove the addresses.json script in favor of this script
   */
  const chains = {}
  Object.entries(addressesJson).forEach(([chainIdStr, contracts]) => {
    const chainId = parseInt(chainIdStr)
    const genesisBlock = chainId === 10 ? OP_GENESIS_BLOCK : 0
    const viemChain = Object.values(viemChains).find(
      (chain) => chain.id === chainId
    )
    const {
      ProxyAdmin,
      OptimismPortalProxy,
      AddressManager,
      L1ERC721BridgeProxy,
      L2OutputOracleProxy,
      L1StandardBridgeProxy,
      L1CrossDomainMessengerProxy,
      OptimismMintableERC20FactoryProxy,
    } = contracts
    /**
     * @type {import('./OpStackChain.js').OpStackChain<number>['contracts']}
     */
    const viemContracts = {
      ...getL2Predeploys(genesisBlock),
      portal: {
        address: OptimismPortalProxy,
      },
      addressManager: {
        address: AddressManager,
      },
      proxyAdmin: {
        address: ProxyAdmin,
      },
      l1ERC721Bridge: {
        address: L1ERC721BridgeProxy,
      },
      l2OutputOracle: {
        address: L2OutputOracleProxy,
      },
      l1StandardBridge: {
        address: L1StandardBridgeProxy,
      },
      l1CrossDomainMessenger: {
        address: L1CrossDomainMessengerProxy,
      },
      l2ERC20Factory: {
        address: OptimismMintableERC20FactoryProxy,
      },
    }
    if (!viemChain) {
      console.warn(
        `no viem chain found for superchain chain ${chainId}! Please notify this chain partner to do a pr to viem`
      )
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
          ...viemContracts,
        },
        // TODO we are assuming an unknown chain has a sourceId of 1 (mainnet)
        sourceId: 1,
      }
    } else {
      chains[chainId] = {
        ...viemChain,
        contracts: {
          ...viemChain.contracts,
          ...viemContracts,
        },
      }
    }
  })

  /**
   * @type {Array<string>}
   */
  const file = []

  Object.values(chains).forEach((chain) => {
    file.push(
      `export const ${camelCase(
        chain.network ?? chain.name
      )} = ${JSON.stringify(chain, null, 2)} as const`
    )
  })

  // EOL
  file.push('\n')

  return file.join('\n')
}
