import { exec } from 'child_process'
import fs from 'fs'
import path from 'path'
import { promisify } from 'util'

const execAsync = promisify(exec)

const outputPath =
  process.argv[2] || path.join(__dirname, '..', '.abi-spec.json')
console.log(`writing abi spec to ${outputPath}`)

const contracts = [
  'src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger',
  'src/L1/L1StandardBridge.sol:L1StandardBridge',
  'src/L1/L2OutputOracle.sol:L2OutputOracle',
  'src/L1/OptimismPortal.sol:OptimismPortal',
  'src/L1/SystemConfig.sol:SystemConfig',
  'src/L1/L1ERC721Bridge.sol:L1ERC721Bridge',
  'src/L1/ProtocolVersions.sol:ProtocolVersions',
  'src/legacy/DeployerWhitelist.sol:DeployerWhitelist',
  'src/L2/L1Block.sol:L1Block',
  'src/legacy/L1BlockNumber.sol:L1BlockNumber',
  'src/L2/L2CrossDomainMessenger.sol:L2CrossDomainMessenger',
  'src/L2/L2StandardBridge.sol:L2StandardBridge',
  'src/L2/L2ToL1MessagePasser.sol:L2ToL1MessagePasser',
  'src/legacy/LegacyERC20ETH.sol:LegacyERC20ETH',
  'src/L2/SequencerFeeVault.sol:SequencerFeeVault',
  'src/L2/BaseFeeVault.sol:BaseFeeVault',
  'src/L2/L1FeeVault.sol:L1FeeVault',
  'src/L2/L2ERC721Bridge.sol:L2ERC721Bridge',
  'src/vendor/WETH9.sol:WETH9',
  'src/universal/ProxyAdmin.sol:ProxyAdmin',
  'src/universal/Proxy.sol:Proxy',
  'src/legacy/L1ChugSplashProxy.sol:L1ChugSplashProxy',
  'src/universal/OptimismMintableERC20.sol:OptimismMintableERC20',
  'src/universal/OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory',
  'src/dispute/DisputeGameFactory.sol:DisputeGameFactory',
]

type ForgeStorageLayoutEntry = {
  label: string
  offset: number
  slot: number
}
type AbiSpecStorageLayoutEntry = {
  slot: number
  offset: number
}
type AbiSpecMethodIdentifiers = { [key: string]: string }
type AbiSpecStorageLayout = { [key: string]: AbiSpecStorageLayoutEntry }
type AbiSpecEntry = {
  methodIdentifiers: AbiSpecMethodIdentifiers
  storageLayout: AbiSpecStorageLayout
}
type AbiSpec = { [key: string]: AbiSpecEntry }

const sortKeys = (obj) => {
  if (typeof obj !== 'object' || obj === null) {
    return obj
  }
  return Object.keys(obj)
    .sort()
    .reduce(
      (acc, key) => {
        acc[key] = sortKeys(obj[key])
        return acc
      },
      Array.isArray(obj) ? [] : {}
    )
}

const getStorageLayout = async (
  contract: string
): Promise<AbiSpecStorageLayout> => {
  const storageLayout: AbiSpecStorageLayout = {}
  const result = await execAsync(`forge inspect ${contract} storage-layout`)
  const layout = JSON.parse(result.stdout)
  const forgeStorageLayout: ForgeStorageLayoutEntry[] = layout['storage']
  for (const entry of forgeStorageLayout) {
    storageLayout[entry.label] = {
      slot: entry.slot,
      offset: entry.offset,
    }
  }
  return storageLayout
}

const getMethodIdentifiers = async (
  contract: string
): Promise<AbiSpecMethodIdentifiers> => {
  const result = await execAsync(`forge inspect ${contract} methodIdentifiers`)
  const ids: { [key: string]: string } = JSON.parse(result.stdout)
  const output: AbiSpecMethodIdentifiers = {}
  for (const [key, value] of Object.entries(ids)) {
    output[value] = key
  }
  return output
}

const main = async () => {
  const spec: AbiSpec = {}

  const storageLayouts = await Promise.all(
    contracts.map((x) => getStorageLayout(x))
  )
  const methodIdentifiersArray = await Promise.all(
    contracts.map((x) => getMethodIdentifiers(x))
  )
  for (let i = 0; i < contracts.length; i++) {
    const toks = contracts[i].split(':')
    const contract = toks[1]
    spec[contract] = {
      methodIdentifiers: methodIdentifiersArray[i],
      storageLayout: storageLayouts[i],
    }
  }

  // consistent sorting for easier diffs
  const output = JSON.stringify(sortKeys(spec), null, 2)
  fs.writeFileSync(outputPath, output)
}

main()
