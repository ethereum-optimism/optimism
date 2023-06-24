import fs from 'fs'
import path from 'path'

import layoutLock from '../layout-lock.json'

/**
 * Directory path to the artifacts.
 * Can be configured as the first argument to the script or
 * defaults to the forge-artifacts directory.
 */
const directoryPath =
  process.argv[2] || path.join(__dirname, '..', 'forge-artifacts')

/**
 * Returns true if the contract should be skipped when inspecting its storage layout.
 * This is useful for abstract contracts that are meant to be inherited.
 * The two particular targets are:
 * - CrossDomainMessengerLegacySpacer0
 * - CrossDomainMessengerLegacySpacer1
 */
const skipped = (contractName: string): boolean => {
  return contractName.includes('CrossDomainMessengerLegacySpacer')
}

/**
 * Parses the fully qualified name of a contract into the name of the contract.
 * For example `contracts/Foo.sol:Foo` becomes `Foo`.
 */
const parseFqn = (name: string): string => {
  const parts = name.split(':')
  return parts[parts.length - 1]
}

/**
 * Parses out variable info from the variable structure in standard compiler json output.
 *
 * @param variable Variable structure from standard compiler json output.
 * @returns Parsed variable info.
 */
const parseVariableInfo = (
  variable: any
): {
  name: string
  slot: number
  offset: number
  length: number
} => {
  // Figure out the length of the variable.
  let variableLength: number
  if (variable.type.startsWith('t_mapping')) {
    variableLength = 32
  } else if (variable.type.startsWith('t_uint')) {
    variableLength = variable.type.match(/uint([0-9]+)/)?.[1] / 8
  } else if (variable.type.startsWith('t_bytes_')) {
    variableLength = 32
  } else if (variable.type.startsWith('t_bytes')) {
    variableLength = variable.type.match(/uint([0-9]+)/)?.[1]
  } else if (variable.type.startsWith('t_address')) {
    variableLength = 20
  } else if (variable.type.startsWith('t_bool')) {
    variableLength = 1
  } else if (variable.type.startsWith('t_array')) {
    // Figure out the size of the type inside of the array
    // and then multiply that by the length of the array.
    // This does not support recursion multiple times for simplicity
    const type = variable.type.match(/^t_array\((\w+)\)/)?.[1]
    const info = parseVariableInfo({
      label: variable.label,
      offset: variable.offset,
      slot: variable.slot,
      type,
    })
    const size = variable.type.match(/^t_array\(\w+\)([0-9]+)/)?.[1]
    variableLength = info.length * parseInt(size, 10)
  } else {
    throw new Error(
      `${variable.label}: unsupported type ${variable.type}, add it to the script`
    )
  }

  return {
    name: variable.label,
    slot: parseInt(variable.slot, 10),
    offset: variable.offset,
    length: variableLength,
  }
}

/**
 * Main logic of the script
 * - Ensures that all of the spacer variables are named correctly
 * - Ensures that storage slots in the layout lock file do not change
 */
const main = async () => {
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

  readFilesRecursively(directoryPath)

  for (const filePath of paths) {
    if (filePath.includes('t.sol')) {
      continue
    }
    const raw = fs.readFileSync(filePath, 'utf8').toString()
    const artifact = JSON.parse(raw)

    // Handle contracts without storage
    const storageLayout = artifact.storageLayout || {}
    if (storageLayout.storage) {
      for (const variable of storageLayout.storage) {
        const fqn = variable.contract
        // Skip some abstract contracts
        if (skipped(fqn)) {
          continue
        }

        const contractName = parseFqn(fqn)

        // Check that the layout lock has not changed
        const lock = layoutLock[contractName] || {}
        if (lock[variable.label]) {
          const variableInfo = parseVariableInfo(variable)
          const expectedInfo = lock[variable.label]
          if (variableInfo.slot !== expectedInfo.slot) {
            throw new Error(`${fqn}.${variable.label} slot has changed`)
          }
          if (variableInfo.offset !== expectedInfo.offset) {
            throw new Error(`${fqn}.${variable.label} offset has changed`)
          }
          if (variableInfo.length !== expectedInfo.length) {
            throw new Error(`${fqn}.${variable.label} length has changed`)
          }
        }

        // Check that the spacers are all named correctly
        if (variable.label.startsWith('spacer_')) {
          const [, slot, offset, length] = variable.label.split('_')
          const variableInfo = parseVariableInfo(variable)

          // Check that the slot is correct.
          if (parseInt(slot, 10) !== variableInfo.slot) {
            throw new Error(
              `${fqn} ${variable.label} is in slot ${variable.slot} but should be in ${slot}`
            )
          }

          // Check that the offset is correct.
          if (parseInt(offset, 10) !== variableInfo.offset) {
            throw new Error(
              `${fqn} ${variable.label} is at offset ${variable.offset} but should be at ${offset}`
            )
          }

          // Check that the length is correct.
          if (parseInt(length, 10) !== variableInfo.length) {
            throw new Error(
              `${fqn} ${variable.label} is ${variableInfo.length} bytes long but should be ${length}`
            )
          }

          console.log(`${fqn}.${variable.label} is valid`)
        }
      }
    }
  }
}

main()
