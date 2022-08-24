import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'

const getFullyQualifiedName = (
  sourceName: string,
  contractName: string
): string => {
  return `${sourceName}:${contractName}`
}

task('inspect')
  .addPositionalParam(
    'contractName',
    'Name of the contract to inspect',
    undefined,
    types.string,
    false
  )
  .addPositionalParam(
    'field',
    'Compiler output field to inspect',
    undefined,
    types.string,
    false
  )
  .setAction(async (args, hre) => {
    const artifact = await hre.artifacts.readArtifact(args.contractName)
    const fqn = getFullyQualifiedName(
      artifact.sourceName,
      artifact.contractName
    )
    const buildInfo = await hre.artifacts.getBuildInfo(fqn)
    const info =
      buildInfo.output.contracts[artifact.sourceName][artifact.contractName]
    if (!info) {
      throw new Error(`Cannot find build info for ${fqn}`)
    }

    try {
      switch (args.field) {
        case 'abi': {
          const abi = info.abi
          console.log(JSON.stringify(abi, null, 2))
          break
        }
        case 'bytecode': {
          const bytecode = info.evm.bytecode.object
          console.log('0x' + bytecode)
          break
        }
        case 'deployedBytecode': {
          const bytecode = info.evm.deployedBytecode.object
          console.log('0x' + bytecode)
          break
        }
        case 'storageLayout': {
          const storageLayout = (info as any).storageLayout
          console.log(JSON.stringify(storageLayout, null, 2))
          break
        }
        case 'methodIdentifiers': {
          const methodIdentifiers = info.evm.methodIdentifiers
          console.log(JSON.stringify(methodIdentifiers, null, 2))
          break
        }
        default: {
          console.log(
            JSON.stringify(
              {
                error: `Unsupported field ${args.field}`,
              },
              null,
              2
            )
          )
          break
        }
      }
    } catch (e) {
      console.log(
        JSON.stringify(
          {
            error: `Cannot find ${args.field}, be sure to enable it in compiler settings`,
          },
          null,
          2
        )
      )
    }
  })
