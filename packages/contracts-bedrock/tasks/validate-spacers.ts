import { task } from 'hardhat/config'

import layoutLock from '../layout-lock.json'

task(
  'validate-spacers',
  'validates that spacer variables are in the correct positions'
).setAction(async (args, hre) => {
  const accounted: string[] = []

  const names = await hre.artifacts.getAllFullyQualifiedNames()
  for (const name of names) {
    // Script is remarkably slow because of getBuildInfo, so better to skip test files since they
    // don't matter for this check.
    if (name.includes('.t.sol')) {
      continue
    }

    // Some files may not have buildInfo (certain libraries). We can safely skip these because we
    // make sure that everything is accounted for anyway.
    const buildInfo = await hre.artifacts.getBuildInfo(name)
    if (buildInfo === undefined) {
      continue
    }

    for (const source of Object.values(buildInfo.output.contracts)) {
      for (const [contractName, contract] of Object.entries(source)) {
        const storageLayout = (contract as any).storageLayout

        // Check that the layout lock is respected.
        if (layoutLock[contractName]) {
          const removed = layoutLock[contractName].filter((variable: any) => {
            return (
              (storageLayout?.storage || []).find(
                (item: any) => item.label === variable
              ) === undefined
            )
          })

          if (removed.length > 0) {
            throw new Error(
              `variable(s) removed from ${contractName}: ${removed.join(', ')}`
            )
          }

          accounted.push(contractName)
        }

        // Check that the positions have not changed.
        for (const variable of storageLayout?.storage || []) {
          if (variable.label.startsWith('spacer_')) {
            const [, slot, offset, length] = variable.label.split('_')

            // Check that the slot is correct.
            if (slot !== variable.slot) {
              throw new Error(
                `${contractName} ${variable.label} is in slot ${variable.slot} but should be in ${slot}`
              )
            }

            // Check that the offset is correct.
            if (parseInt(offset, 10) !== variable.offset) {
              throw new Error(
                `${contractName} ${variable.label} is at offset ${variable.offset} but should be at ${offset}`
              )
            }

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
            } else {
              throw new Error('unsupported type, add it to the script')
            }

            // Check that the length is correct.
            if (parseInt(length, 10) !== variableLength) {
              throw new Error(
                `${contractName} ${variable.label} is ${variableLength} bytes long but should be ${length}`
              )
            }
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
