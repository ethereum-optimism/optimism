import fs from 'fs'
import path from 'path'

const outdir = process.argv[2] || path.join(__dirname, '..', 'snapshots')
const forgeArtifactsDir = path.join(__dirname, '..', 'forge-artifacts')

const getAllContractsSources = (): Array<string> => {
  const paths = []
  const readFilesRecursively = (dir: string) => {
    const files = fs.readdirSync(dir)

    for (const file of files) {
      const filePath = path.join(dir, file)
      const fileStat = fs.statSync(filePath)

      if (fileStat.isDirectory()) {
        readFilesRecursively(filePath)
      } else {
        paths.push(filePath)
      }
    }
  }
  readFilesRecursively(path.join(__dirname, '..', 'src'))

  return paths
    .filter((x) => x.endsWith('.sol'))
    .map((p: string) => path.basename(p))
    .sort()
}

type ForgeArtifact = {
  abi: object
  ast: {
    nodeType: string
    nodes: any[]
  }
  storageLayout: {
    storage: [{ type: string; label: string; offset: number; slot: number }]
    types: { [key: string]: { label: string; numberOfBytes: number } }
  }
  bytecode: {
    object: string
  }
}

type AbiSpecStorageLayoutEntry = {
  label: string
  slot: number
  offset: number
  bytes: number
  type: string
}
const sortKeys = (obj: any) => {
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

// ContractName.0.9.8.json -> ContractName.sol
// ContractName.json -> ContractName.sol
const parseArtifactName = (artifactVersionFile: string): string => {
  const match = artifactVersionFile.match(/(.*?)\.([0-9]+\.[0-9]+\.[0-9]+)?/)
  if (!match) {
    throw new Error(`Invalid artifact file name: ${artifactVersionFile}`)
  }
  return match[1]
}

const main = async () => {
  console.log(`writing abi and storage layout snapshots to ${outdir}`)

  const storageLayoutDir = path.join(outdir, 'storageLayout')
  const abiDir = path.join(outdir, 'abi')
  fs.mkdirSync(storageLayoutDir, { recursive: true })
  fs.mkdirSync(abiDir, { recursive: true })

  const contractSources = getAllContractsSources()
  const knownAbis = {}

  for (const contractFile of contractSources) {
    const contractArtifacts = path.join(forgeArtifactsDir, contractFile)
    for (const name of fs.readdirSync(contractArtifacts)) {
      const data = fs.readFileSync(path.join(contractArtifacts, name))
      const artifact: ForgeArtifact = JSON.parse(data.toString())

      const contractName = parseArtifactName(name)

      // HACK: This is a hack to ignore libraries and abstract contracts. Not robust against changes to solc's internal ast repr
      const isContract = artifact.ast.nodes.some((node: any) => {
        return (
          node.nodeType === 'ContractDefinition' &&
          node.name === contractName &&
          node.contractKind === 'contract' &&
          (node.abstract === undefined || // solc < 0.6 doesn't have explicit abstract contracts
            node.abstract === false)
        )
      })
      if (!isContract) {
        console.log(`ignoring library/interface ${contractName}`)
        continue
      }

      const storageLayout: AbiSpecStorageLayoutEntry[] = []
      for (const storageEntry of artifact.storageLayout.storage) {
        // convert ast-based type to solidity type
        const typ = artifact.storageLayout.types[storageEntry.type]
        if (typ === undefined) {
          throw new Error(
            `undefined type for ${contractName}:${storageEntry.label}`
          )
        }
        storageLayout.push({
          label: storageEntry.label,
          bytes: typ.numberOfBytes,
          offset: storageEntry.offset,
          slot: storageEntry.slot,
          type: typ.label,
        })
      }

      if (knownAbis[contractName] === undefined) {
        knownAbis[contractName] = artifact.abi
      } else if (
        JSON.stringify(knownAbis[contractName]) !== JSON.stringify(artifact.abi)
      ) {
        throw Error(
          `detected multiple artifact versions with different ABIs for ${contractFile}`
        )
      } else {
        console.log(`detected multiple artifacts for ${contractName}`)
      }

      // Sort snapshots for easier manual inspection
      fs.writeFileSync(
        `${abiDir}/${contractName}.json`,
        JSON.stringify(sortKeys(artifact.abi), null, 2)
      )
      fs.writeFileSync(
        `${storageLayoutDir}/${contractName}.json`,
        JSON.stringify(sortKeys(storageLayout), null, 2)
      )
    }
  }
}

main()
