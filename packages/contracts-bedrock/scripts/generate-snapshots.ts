import fs from 'fs'
import path from 'path'

const outdir = process.argv[2] || path.join(__dirname, '..', 'snapshots')
const forgeArtifactsDir = path.join(__dirname, '..', 'forge-artifacts')

const getAllContracts = (): Array<string> => {
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

  // Assumes there is a single contract per file
  return paths
    .filter((x) => x.endsWith('.sol'))
    .map((p: string) => {
      const b = path.basename(p)
      return `${b}:${b.replace('.sol', '')}`
    })
    .sort()
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
  abi: any
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

const main = async () => {
  console.log(`writing abi spec to ${outdir}`)
  fs.mkdirSync(outdir, { recursive: true })

  const contracts = getAllContracts()

  for (const contract of contracts) {
    const toks = contract.split(':')
    const contractFile = contract.split(':')[0]
    const contractName = toks[1]

    const storageLayout: AbiSpecStorageLayout = {}

    let artifactFile = path.join(
      forgeArtifactsDir,
      contractFile,
      `${contractName}.json`
    )

    // NOTE: Read the first version in the directory. We may want to assert that all version's ABIs are identical
    if (!fs.existsSync(artifactFile)) {
      const filename = fs.readdirSync(path.dirname(artifactFile))[0]
      artifactFile = path.join(path.dirname(artifactFile), filename)
    }

    const data = fs.readFileSync(artifactFile)
    const artifact = JSON.parse(data.toString())

    // ignore abstract contracts
    if (artifact.bytecode.object === '0x') {
      console.log(`ignoring interface ${contractName}`)
      continue
    }

    // HACK: This is a hack to ignore libraries. Not robust against changes to solc's internal ast repr
    const isContract = artifact.ast.nodes.find((node: any) => {
      if (
        node.nodeType === 'ContractDefinition' &&
        node.name === contractName &&
        node.kind === 'contract'
      ) {
        return node
      }
    })
    if (!isContract) {
      console.log(`ignoring library/interface ${contractName}`)
      continue
    }

    for (const storageEntry of artifact.storageLayout.storage) {
      storageLayout[storageEntry.label] = {
        slot: storageEntry.slot,
        offset: storageEntry.offset,
      }
    }
    const ids: AbiSpecMethodIdentifiers = {}
    for (const [key, value] of Object.entries(artifact.methodIdentifiers)) {
      ids[value as string] = key
    }

    const entry: AbiSpecEntry = {
      methodIdentifiers: ids,
      storageLayout,
      abi: artifact.abi,
    }
    fs.writeFileSync(
      `${outdir}/${contractName}.json`,
      JSON.stringify(sortKeys(entry), null, 2)
    )
  }
}

main()
