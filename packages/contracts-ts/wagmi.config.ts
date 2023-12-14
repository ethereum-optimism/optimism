import { ContractConfig, defineConfig, Plugin } from '@wagmi/cli'
import { actions, react } from '@wagmi/cli/plugins'
import * as glob from 'glob'
import { readFileSync, writeFileSync } from 'fs'
import type { Abi, Address } from 'abitype'
import { isDeepStrictEqual } from 'util'
import { camelCase, constantCase } from 'change-case'

/**
 * Predeployed contract addresses
 * In future it would be nice to have a json file in contracts bedrock be generated as source of truth
 * Keep this in sync with op-bindings/predeploys/addresses.go in meantime
 */
const predeployContracts = {
  LegacyMessagePasser: {
    address: '0x4200000000000000000000000000000000000000',
    introduced: 'Legacy',
    deprecated: true,
    proxied: true,
  },
  DeployerWhitelist: {
    address: '0x4200000000000000000000000000000000000002',
    introduced: 'Legacy',
    deprecated: true,
    proxied: true,
  },
  LegacyERC20ETH: {
    address: '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',
    introduced: 'Legacy',
    deprecated: true,
    proxied: false,
  },
  WETH9: {
    address: '0x4200000000000000000000000000000000000006',
    introduced: 'Legacy',
    deprecated: false,
    proxied: false,
  },
  L2CrossDomainMessenger: {
    address: '0x4200000000000000000000000000000000000007',
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  L2StandardBridge: {
    address: '0x4200000000000000000000000000000000000010',
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  SequencerFeeVault: {
    address: '0x4200000000000000000000000000000000000011',
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  OptimismMintableERC20Factory: {
    address: '0x4200000000000000000000000000000000000012',
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  L1BlockNumber: {
    address: '0x4200000000000000000000000000000000000013',
    introduced: 'Legacy',
    deprecated: true,
    proxied: true,
  },
  GasPriceOracle: {
    address: '0x420000000000000000000000000000000000000F',
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  GovernanceToken: {
    address: '0x4200000000000000000000000000000000000042',
    introduced: 'Legacy',
    deprecated: false,
    proxied: false,
  },
  L1Block: {
    address: '0x4200000000000000000000000000000000000015',
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  L2ToL1MessagePasser: {
    address: '0x4200000000000000000000000000000000000016',
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  L2ERC721Bridge: {
    address: '0x4200000000000000000000000000000000000014',
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  OptimismMintableERC721Factory: {
    address: '0x4200000000000000000000000000000000000017',
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  ProxyAdmin: {
    address: '0x4200000000000000000000000000000000000018',
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  BaseFeeVault: {
    address: '0x4200000000000000000000000000000000000019',
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  L1FeeVault: {
    address: '0x420000000000000000000000000000000000001a',
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
} as const

type DeploymentJson = {
  abi: Abi
  address: `0x${string}`
}
const chains = {
  1: 'mainnet',
  10: 'optimism-mainnet',
  5: 'goerli',
  420: 'optimism-goerli',
} as const

if (!glob.sync('node_modules/*').length) {
  throw new Error(
    'No node_modules found. Please run `pnpm install` before running this script'
  )
}

const deployments = {
  [1]: glob.sync(
    `node_modules/@eth-optimism/contracts-bedrock/deployments/${chains[1]}/*.json`
  ),
  [10]: glob.sync(
    `node_modules/@eth-optimism/contracts-bedrock/deployments/${chains[10]}/*.json`
  ),
  [5]: glob.sync(
    `node_modules/@eth-optimism/contracts-bedrock/deployments/${chains[5]}/*.json`
  ),
  [420]: glob.sync(
    `node_modules/@eth-optimism/contracts-bedrock/deployments/${chains[420]}/*.json`
  ),
}
Object.entries(deployments).forEach(([chain, deploymentFiles]) => {
  if (deploymentFiles.length === 0) {
    throw new Error(`No bedrock deployments found for ${chains[chain]}`)
  }
})

const getWagmiContracts = (
  deploymentFiles: string[],
  filterDuplicates = false
) =>
  deploymentFiles.map((artifactPath) => {
    const deployment = JSON.parse(
      readFileSync(artifactPath, 'utf8')
    ) as DeploymentJson

    // There is a known bug in the wagmi/cli repo where some contracts have FOO_CASE and fooCase in same contract causing issues
    // This is a common pattern at OP
    // @see https://github.com/wagmi-dev/wagmi/issues/2724
    const abi = filterDuplicates
      ? deployment.abi.filter((item) => {
          if (item.type !== 'function') {
            return true
          }
          if (item.name !== constantCase(item.name)) {
            return true
          }
          // if constante case make sure it is not a duplicate
          // e.g. make sure fooBar doesn't exist with FOO_BAR
          return !deployment.abi.some(
            (otherItem) =>
              otherItem.type === 'function' &&
              otherItem.name !== item.name &&
              otherItem.name === camelCase(item.name)
          )
        })
      : deployment.abi
    const contractConfig = {
      abi,
      name: artifactPath.split('/').reverse()[0]?.replace('.json', ''),
      address: deployment.address,
    } satisfies ContractConfig
    if (!contractConfig.name) {
      throw new Error(
        'Unable to identify the name of the contract at ' + artifactPath
      )
    }
    return contractConfig
  })

/**
 * Returns the contracts for the wagmi cli config
 */
const getContractConfigs = (filterDuplicates = false) => {
  const contracts = {
    1: getWagmiContracts(deployments[1], filterDuplicates),
    10: getWagmiContracts(deployments[10], filterDuplicates),
    5: getWagmiContracts(deployments[5], filterDuplicates),
    420: getWagmiContracts(deployments[420], filterDuplicates),
  }

  const allContracts = Object.values(contracts).flat()

  const config: ContractConfig[] = []

  // this for loop is not terribly efficient but seems fast enough for the scale here
  for (const contract of allContracts) {
    // we will only process the implementation ABI but will use teh proxy addresses for deployments
    const isProxy = contract.name.endsWith('Proxy')
    // once we see the first deployment of a contract we will process all networks all at once
    const alreadyProcessedContract = config.find(
      (c) => c.name === contract.name
    )
    if (isProxy || alreadyProcessedContract) {
      continue
    }

    const implementations = {
      // @warning Later code assumes mainnet is first!!!
      [1]: contracts[1].find((c) => c.name === contract.name),
      // @warning Later code assumes mainnet is first!!!
      [10]: contracts[10].find((c) => c.name === contract.name),
      [5]: contracts[5].find((c) => c.name === contract.name),
      [420]: contracts[420].find((c) => c.name === contract.name),
    }
    const maybeProxyName = contract.name + 'Proxy'
    const proxies = {
      // @warning Later code assumes mainnet is first!!!
      [1]: contracts[1].find((c) => c.name === maybeProxyName),
      // @warning Later code assumes mainnet is first!!!
      [10]: contracts[10].find((c) => c.name === maybeProxyName),
      [5]: contracts[5].find((c) => c.name === maybeProxyName),
      [420]: contracts[420].find((c) => c.name === maybeProxyName),
    }

    const predeploy = predeployContracts[
      contract.name as keyof typeof predeployContracts
    ] as { address: Address } | undefined

    // If the contract has different abis on different networks we don't want to group them as a single abi
    const isContractUnique = !Object.values(implementations).some(
      (implementation) =>
        implementation && !isDeepStrictEqual(implementation.abi, contract.abi)
    )
    if (!isContractUnique) {
      Object.entries(implementations)
        .filter(([_, implementation]) => implementation)
        .forEach(([chain, implementation], i) => {
          if (implementation) {
            // make the first one canonical.  This will be mainnet or op mainnet if they exist
            const name =
              i === 0 ? contract.name : `${contract.name}_${chains[chain]}`
            const nextConfig = {
              abi: implementation.abi,
              name,
              address: {
                [Number.parseInt(chain)]:
                  predeploy?.address ??
                  proxies[chain]?.address ??
                  implementation?.address,
              }, // predeploy?.address ?? proxies[chain]?.address ?? implementation?.address
            } satisfies ContractConfig
            config.push(nextConfig)
          }
        })
      continue
    }

    const wagmiConfig = {
      abi: contract.abi,
      name: contract.name,
      address: {},
    } satisfies ContractConfig

    Object.entries(implementations).forEach(([chain, proxy]) => {
      if (proxy) {
        wagmiConfig.address[chain] =
          predeploy?.address ?? proxy.address ?? contract.address
      }
    })
    // if proxies exist overwrite the address with the proxy address
    Object.entries(proxies).forEach(([chain, proxy]) => {
      if (proxy) {
        wagmiConfig.address[chain] = predeploy?.address ?? proxy.address
      }
    })

    config.push(wagmiConfig)
  }

  return config
}

/**
 * This plugin will create a addresses mapping from contract name to address
 */
const addressesByContractByNetworkPlugin: Plugin = {
  name: 'addressesByContractByNetwork',
  run: async ({ contracts }) => {
    const addresses = Object.fromEntries(
      contracts.map((contract) => [contract.name, contract.address ?? {}])
    )
    // write to json file so it's easy to audit in prs relative to the generated file diff
    writeFileSync('./addresses.json', JSON.stringify(addresses, null, 2))
    return {
      content: [
        `export const addresses = ${JSON.stringify(addresses)} as const`,
        `export const predeploys = ${JSON.stringify(predeployContracts)}`,
      ].join('\n'),
    }
  },
}

/**
 * This plugin will create an abi mapping from contract name to abi
 */
const abiPlugin: Plugin = {
  name: 'abisByContractByNetwork',
  run: async ({ contracts }) => {
    const abis = Object.fromEntries(
      contracts.map((contract) => [contract.name, contract.abi])
    )
    // write to json file so it's easy to audit in prs relative to the generated file diff
    writeFileSync('./abis.json', JSON.stringify(abis, null, 2))
    return {
      content: `export const abis = ${JSON.stringify(abis)} as const`,
    }
  },
}

/**
 * This plugin adds an eslint ignore to the generated code
 */
const eslintIgnorePlugin: Plugin = {
  name: 'eslintIgnore',
  run: async () => {
    return {
      prepend: `/* eslint-disable */`,
      content: ``,
    }
  },
}

const contracts = getContractConfigs()
// there is a known wagmi bug with contracts who have both FOO_BAR and fooBar method
const contractsWithFilteredDuplicates = getContractConfigs(true)
// @see https://wagmi.sh/cli
export default defineConfig([
  {
    out: 'src/constants.ts',
    contracts,
    plugins: [
      eslintIgnorePlugin,
      addressesByContractByNetworkPlugin,
      abiPlugin,
    ],
  },
  {
    out: 'src/actions.ts',
    contracts: contractsWithFilteredDuplicates,
    plugins: [
      eslintIgnorePlugin,
      actions({
        getContract: true,
        // don't include actions because they can be more simply done via getContract
        prepareWriteContract: false,
        readContract: false,
        watchContractEvent: false,
        writeContract: false,
      }),
    ],
  },
  {
    out: 'src/react.ts',
    contracts: contractsWithFilteredDuplicates,
    plugins: [
      eslintIgnorePlugin,
      react({
        useContractRead: true,
        useContractWrite: true,
        useContractEvent: true,
        // don't include more niche actions to keep api more simple
        useContractFunctionRead: false,
        useContractFunctionWrite: false,
        useContractItemEvent: false,
        usePrepareContractFunctionWrite: false,
        usePrepareContractWrite: false,
      }),
    ],
  },
])
