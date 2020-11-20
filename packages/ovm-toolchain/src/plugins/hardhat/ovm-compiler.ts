import { subtask } from 'hardhat/config'
import {
  TASK_COMPILE_SOLIDITY_RUN_SOLCJS,
  TASK_COMPILE_SOLIDITY_RUN_SOLC,
} from 'hardhat/builtin-tasks/task-names'

subtask(
  TASK_COMPILE_SOLIDITY_RUN_SOLC,
  async (
    { input, solcPath }: { input: any; solcPath: string },
    { config, run },
    runSuper
  ) => {
    // Try to find a path to @eth-optimism/solc, throw if we can't.
    let ovmSolcPath: string
    try {
      ovmSolcPath = require.resolve('@eth-optimism/solc/soljson.js')
    } catch (err) {
      if (err.toString().contains('Cannot find module')) {
        throw new Error(
          `ovm-toolchain: Could not find "@eth-optimism/solc" in your node_modules.`
        )
      } else {
        throw err
      }
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
    const re = new RegExp('// +build (.*?)$', 'gm')
    for (const file of Object.keys(input.sources)) {
      const content = input.sources[file].content
      const matches = content.match(re)

      if (
        matches &&
        matches.some((match: string) => {
          return match.includes('ovm')
        })
      ) {
        ovmInput.sources[file] = input.sources[file]
      } else {
        evmInput.sources[file] = input.sources[file]
      }
    }

    // Build both inputs separately.
    const ovmOutput = await run(TASK_COMPILE_SOLIDITY_RUN_SOLCJS, {
      input: ovmInput,
      solcJsPath: ovmSolcPath,
    })
    const evmOutput = await runSuper({ input: evmInput, solcPath })

    // Filter out any "No input sources specified" errors, but only if one of the two compilations
    // threw the error.
    let errors = (ovmOutput.errors || []).concat(evmOutput.errors || [])
    const filtered = errors.filter((error: any) => {
      return error.message !== 'No input sources specified.'
    })
    if (errors.length === filtered.length + 1) {
      errors = filtered
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
