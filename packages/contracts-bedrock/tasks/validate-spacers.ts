import { task } from 'hardhat/config'
import { parseFullyQualifiedName } from 'hardhat/utils/contract-names'
import { HardhatRuntimeEnvironment } from 'hardhat/types'

import layoutLock from '../layout-lock.json'

// Artifacts that should be skipped when inspecting their storage layout
const skipped = {
  // Both of these are skipped because they are meant to be inherited
  // by the CrossDomainMessenger. It is the CrossDomainMessenger that
  // should be inspected, not these contracts.
  CrossDomainMessengerLegacySpacer0: true,
  CrossDomainMessengerLegacySpacer1: true,
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

task(
  'validate-spacers',
  'validates that spacer variables are in the correct positions'
).setAction(async ({}, hre: HardhatRuntimeEnvironment) => {
  const accounted: string[] = []

  const names = await hre.artifacts.getAllFullyQualifiedNames()
  for (const fqn of names) {
    // Script is remarkably slow because of getBuildInfo, so better to skip test files since they
    // don't matter for this check.
    if (fqn.includes('.t.sol')) {
      continue
    }

    console.log(`Processing ${fqn}`)
    const parsed = parseFullyQualifiedName(fqn)
    const contractName = parsed.contractName

    if (skipped[contractName] === true) {
      console.log(`Skipping ${contractName} because it is marked as skippable`)
      continue
    }

    // Some files may not have buildInfo (certain libraries). We can safely skip these because we
    // make sure that everything is accounted for anyway.
    const buildInfo = await hre.artifacts.getBuildInfo(fqn)
    if (buildInfo === undefined) {
      console.log(`Skipping ${fqn} because it has no buildInfo`)
      continue
    }

    const sources = buildInfo.output.contracts
    for (const [sourceName, source] of Object.entries(sources)) {
      // The source file may have our contract
      if (sourceName.includes(parsed.sourceName)) {
        const contract = source[contractName]
        if (!contract) {
          console.log(`Skipping ${contractName} as its not found in the source`)
          continue
        }
        const storageLayout = (contract as any).storageLayout
        if (!storageLayout) {
          continue
        }

        if (layoutLock[contractName]) {
          console.log(`Processing layout lock for ${contractName}`)
          const removed = Object.entries(layoutLock[contractName]).filter(
            ([key, val]: any) => {
              const storage = storageLayout?.storage || []
              return !storage.some((item: any) => {
                // Skip anything that doesn't clearly match the key because otherwise we'll get an
                // error while parsing the variable info for unsupported variable types.
                if (!item.label.includes(key)) {
                  return false
                }

                // Make sure the variable matches **exactly**.
                const variableInfo = parseVariableInfo(item)

                return (
                  variableInfo.name === key &&
                  variableInfo.offset === val.offset &&
                  variableInfo.slot === val.slot &&
                  variableInfo.length === val.length
                )
              })
            }
          )

          if (removed.length > 0) {
            throw new Error(
              `variable(s) removed from ${contractName}: ${removed.join(', ')}`
            )
          }
          console.log(`Valid layout lock for ${contractName}`)
          accounted.push(contractName)
        }

        for (const variable of storageLayout?.storage || []) {
          if (variable.label.startsWith('spacer_')) {
            const [, slot, offset, length] = variable.label.split('_')
            const variableInfo = parseVariableInfo(variable)

            // Check that the slot is correct.
            if (parseInt(slot, 10) !== variableInfo.slot) {
              throw new Error(
                `${contractName} ${variable.label} is in slot ${variable.slot} but should be in ${slot}`
              )
            }

            // Check that the offset is correct.
            if (parseInt(offset, 10) !== variableInfo.offset) {
              throw new Error(
                `${contractName} ${variable.label} is at offset ${variable.offset} but should be at ${offset}`
              )
            }

            // Check that the length is correct.
            if (parseInt(length, 10) !== variableInfo.length) {
              throw new Error(
                `${contractName} ${variable.label} is ${variableInfo.length} bytes long but should be ${length}`
              )
            }

            console.log(`${contractName}.${variable.label} is valid`)
          }
        }
      }
    }
  }

  for (const name of Object.keys(layoutLock)) {
    if (!accounted.includes(name)) {
      throw new Error(`contract ${name} is not accounted for`)
    }
  }
})
