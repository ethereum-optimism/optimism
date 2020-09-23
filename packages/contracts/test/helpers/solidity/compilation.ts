import * as path from 'path'
import bre from '@nomiclabs/buidler'
import { Compiler } from '@nomiclabs/buidler/internal/solidity/compiler'

export interface SolidityCompiler {
  version: () => string
  compile: any
}

export interface ContractSource {
  path: string
  content: string
}

export const getDefaultCompiler = async (): Promise<SolidityCompiler> => {
  const compiler = new Compiler(
    bre.config.solc.version,
    path.join(bre.config.paths.cache, 'compilers')
  )

  return compiler.getSolc()
}

export const compile = async (
  sources: ContractSource[],
  compiler?: SolidityCompiler
): Promise<any> => {
  compiler = compiler || (await getDefaultCompiler())

  const compilerInput = {
    language: 'Solidity',
    sources: sources.reduce((parsed, source) => {
      parsed[source.path] = {
        content: source.content,
      }

      return parsed
    }, {}),
    settings: {
      outputSelection: {
        '*': {
          '*': ['*'],
        },
      },
    },
  }

  return JSON.parse(compiler.compile(JSON.stringify(compilerInput)))
}
