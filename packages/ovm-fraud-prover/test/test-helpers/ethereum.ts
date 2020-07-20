/* External Imports */
import * as path from 'path'
import * as fs from 'fs'
import * as transpiler from '@eth-optimism/solc-transpiler'

interface ContractSource {
  [contract: string]: {
    content: string
  }
}

/**
 * Simple function for compiling Solidity contracts without frills.
 * @param compiler Solc compiler to use in compilation.
 * @param file Contract source or path to compile.
 * @param settings Extra compiler settings.
 * @returns Compiled contract JSON.
 */
export const compile = (
  compiler: any,
  file: string,
  settings: any = {}
): any => {
  const isFilePath = fs.existsSync(file)

  let contractSource: ContractSource
  let contractName: string
  if (isFilePath) {
    contractName = path.basename(file)
    contractSource = {
      [contractName]: {
        content: fs.readFileSync(file, 'utf8'),
      }
    }
  } else {
    const regexp = new RegExp('(?<=(?:^|\n|\n\r))(?:contract|library) (.*?) {', 'g')
    contractName = regexp.exec(file)[0] + '.sol'

    contractSource = {
      [contractName]: {
        content: file
      }
    }
  }

  const input = {
    language: 'Solidity',
    sources: {
      ...contractSource
    },
    settings: {
      outputSelection: {
        '*': {
          '*': ['*'],
        },
      },
      ...settings,
    },
  }

  return JSON.parse(compiler.compile(JSON.stringify(input))).contracts[contractName]
}

/**
 * Compiles a contract using the OVM transpiler.
 * @param file Contract source or path to compile.
 * @param executionManagerAddress Address of the execution manager.
 * @param settings Extra settings to the compiler.
 * @returns Transpiled contact JSON.
 */
export const transpile = (
  file: string,
  executionManagerAddress: string,
  settings: any = {}
): any => {
  return compile(transpiler, file, {
    ...settings,
    ...{
      executionManagerAddress,
    }
  })
}