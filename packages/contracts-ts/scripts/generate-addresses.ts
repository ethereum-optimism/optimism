import glob from 'glob'
import fs from 'fs'
import path from 'path'
import { z } from 'zod'
import { Abi as AbiValidator, Address as AddressValidator } from 'abitype/zod'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

/**
 * [zod](https://github.com/colinhacks/zod) validator for how the deployments json files
 * are expected to be shaped
 */
const deploymentValidator = z.object({
  address: AddressValidator,
  abi: AbiValidator,
})
type Deployment = z.infer<typeof deploymentValidator>
type Address = z.infer<typeof AddressValidator>

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

const nodeModules = path.join(__dirname, '..', 'node_modules')

const deployments = {
  [1]: glob.sync(
    path.join(
      nodeModules,
      '@eth-optimism',
      'contracts-bedrock',
      'deployments',
      chains[1],
      '*.json'
    )
  ),
  [10]: glob.sync(
    path.join(
      nodeModules,
      '@eth-optimism',
      'contracts-bedrock',
      'deployments',
      chains[10],
      '*.json'
    )
  ),
  [5]: glob.sync(
    path.join(
      nodeModules,
      '@eth-optimism',
      'contracts-bedrock',
      'deployments',
      chains[5],
      '*.json'
    )
  ),
  [420]: glob.sync(
    path.join(
      nodeModules,
      '@eth-optimism',
      'contracts-bedrock',
      'deployments',
      chains[420],
      '*.json'
    )
  ),
}

Object.entries(deployments).forEach(([chain, deploymentFiles]) => {
  if (deploymentFiles.length === 0) {
    throw new Error(`No bedrock deployments found for ${chains[chain]}`)
  }
})

const getArtifacts = async () => {
  const artifactPromises: Promise<
    Deployment & { chainId: number; contractName: string }
  >[] = []
  // same loop as 2 for loops
  for (const [chainId, deploymentFiles] of Object.entries(deployments)) {
    for (const artifactPath of deploymentFiles) {
      const contractName = artifactPath
        .split('/')
        .reverse()[0]
        ?.replace('.json', '')
      const artifact = fs.promises
        .readFile(artifactPath, 'utf8')
        .then((a) => deploymentValidator.parse(JSON.parse(a)))
        .then((deployment) => ({
          ...deployment,
          chainId: Number.parseInt(chainId),
          contractName,
        }))
      artifactPromises.push(artifact)
    }
  }
  return Promise.all(artifactPromises)
}

const generate = async () => {
  const artifacts = await getArtifacts()

  const addresses: {
    [contractName: string]: {
      [chainId: number]: Address
    }
  } = {}

  for (const [contractName, { address }] of Object.entries(
    predeployContracts
  )) {
    for (const chainId of Object.keys(chains)) {
      addresses[contractName] = {
        ...addresses[contractName],
        [Number.parseInt(chainId)]: address,
      }
    }
  }

  for (const artifact of artifacts) {
    if (addresses[artifact.contractName]?.[artifact.chainId]) {
      console.warn(
        `Duplicate artifact found for ${artifact.contractName} on chain ${
          artifact.chainId
        } at addresses ${
          addresses[artifact.contractName][artifact.chainId]
        } and ${artifact.address}}. Using ${
          addresses[artifact.contractName][artifact.chainId]
        }`
      )
      continue
    }
    addresses[artifact.contractName] = {
      ...addresses[artifact.contractName],
      [artifact.chainId]: artifact.address,
    }
  }

  await fs.promises.writeFile(
    process.argv[2] || path.join(__dirname, '..', 'addresses.json'),
    JSON.stringify(addresses, null, 2)
  )
}

generate()
