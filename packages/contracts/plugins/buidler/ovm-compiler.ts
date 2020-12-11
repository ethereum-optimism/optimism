import * as path from 'path'
import fsExtra from 'fs-extra'
import { internalTask } from '@nomiclabs/buidler/config'
import { SolcInput } from '@nomiclabs/buidler/types'
import { Compiler } from '@nomiclabs/buidler/internal/solidity/compiler'
import { pluralize } from '@nomiclabs/buidler/internal/util/strings'
import {
  saveArtifact,
  getArtifactFromContractOutput,
} from '@nomiclabs/buidler/internal/artifacts'
import {
  TASK_COMPILE_RUN_COMPILER,
  TASK_BUILD_ARTIFACTS,
  TASK_COMPILE_GET_SOURCE_PATHS,
  TASK_COMPILE_CHECK_CACHE,
  TASK_COMPILE_COMPILE,
  TASK_COMPILE_GET_COMPILER_INPUT,
} from '@nomiclabs/buidler/builtin-tasks/task-names'

internalTask(TASK_COMPILE_RUN_COMPILER).setAction(
  async ({ input }: { input: SolcInput }, { config }) => {
    // Try to find a path to @eth-optimism/solc, throw if we can't.
    let ovmSolcJs: any
    try {
      ovmSolcJs = require('@eth-optimism/solc')
    } catch (err) {
      if (err.toString().contains('Cannot find module')) {
        throw new Error(
          `ovm-toolchain: Could not find "@eth-optimism/solc" in your node_modules.`
        )
      } else {
        throw err
      }
    }

    const evmCompiler = new Compiler(
      config.solc.version,
      path.join(config.paths.cache, 'compilers')
    )

    const ovmCompiler = new Compiler(
      ovmSolcJs.version(),
      path.join(config.paths.cache, 'compilers')
    )

    ovmCompiler.getSolc = () => {
      return ovmSolcJs
    }

    const ovmInput = {
      language: 'Solidity',
      sources: {},
      settings: input.settings,
    }
    const evmInput = {
      language: 'Solidity',
      sources: {},
      settings: input.settings,
    }

    // Separate the EVM and OVM inputs.
    for (const file of Object.keys(input.sources)) {
      evmInput.sources[file] = input.sources[file]

      if (input.sources[file].content.includes('// +build ovm')) {
        ovmInput.sources[file] = input.sources[file]
      }
    }

    // Build both inputs separately.
    console.log('Compiling ovm contracts...')
    const ovmOutput = await ovmCompiler.compile(ovmInput)
    console.log('Compiling evm contracts...')
    const evmOutput = await evmCompiler.compile(evmInput)

    // Filter out any "No input sources specified" errors, but only if one of the two compilations
    // threw the error.
    let errors = (ovmOutput.errors || []).concat(evmOutput.errors || [])
    const filtered = errors.filter((error: any) => {
      return error.message !== 'No input sources specified.'
    })
    if (errors.length === filtered.length + 1) {
      errors = filtered
    }

    for (const name of Object.keys(ovmOutput.contracts)) {
      ovmOutput.contracts[`${name}.ovm`] = ovmOutput.contracts[name]
      delete ovmOutput.contracts[name]
    }

    // Combine the outputs.
    const output = {
      contracts: {
        ...ovmOutput.contracts,
        ...evmOutput.contracts,
      },
      errors,
      sources: {
        ...ovmOutput.sources,
        ...evmOutput.sources,
      },
    }

    return output
  }
)

internalTask(
  TASK_COMPILE_GET_COMPILER_INPUT,
  async (_, { config, run }, runSuper) => {
    const input = await runSuper()

    // For smock.
    input.settings.outputSelection['*']['*'].push('storageLayout')

    return input
  }
)

internalTask(TASK_BUILD_ARTIFACTS, async ({ force }, { config, run }) => {
  const sources = await run(TASK_COMPILE_GET_SOURCE_PATHS)

  if (sources.length === 0) {
    console.log('No Solidity source file available.')
    return
  }

  const isCached: boolean = await run(TASK_COMPILE_CHECK_CACHE, { force })

  if (isCached) {
    console.log(
      'All contracts have already been compiled, skipping compilation.'
    )
    return
  }

  const compilationOutput = await run(TASK_COMPILE_COMPILE)

  if (compilationOutput === undefined) {
    return
  }

  await fsExtra.ensureDir(config.paths.artifacts)
  let numberOfContracts = 0

  for (const [fileName, file] of Object.entries<any>(
    compilationOutput.contracts
  )) {
    for (const [contractName, contractOutput] of Object.entries(file)) {
      const artifact = getArtifactFromContractOutput(
        contractName,
        contractOutput
      )
      numberOfContracts += 1

      // For smock.
      ;(artifact as any).storageLayout = (contractOutput as any).storageLayout

      if (fileName.endsWith('.ovm')) {
        await saveArtifact(config.paths.artifacts + '/ovm', artifact)
      } else {
        await saveArtifact(config.paths.artifacts, artifact)
      }
    }
  }

  console.log(
    'Compiled',
    numberOfContracts,
    pluralize(numberOfContracts, 'contract'),
    'successfully'
  )
})
