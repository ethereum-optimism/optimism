import { task } from 'hardhat/config'

task(
  'validate-spacers',
  'validates that spacer variables are in the correct positions'
).setAction(async (args, hre) => {
  const names = await hre.artifacts.getAllFullyQualifiedNames()
  for (const name of names) {
    // Script is remarkably slow because of getBuildInfo, so better to skip test files since they
    // don't matter for this check.
    if (name.includes('.t.sol')) {
      continue
    }

    const buildInfo = await hre.artifacts.getBuildInfo(name)
    for (const source of Object.values(buildInfo.output.contracts)) {
      for (const [contractName, contract] of Object.entries(source)) {
        const storageLayout = (contract as any).storageLayout
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
})
