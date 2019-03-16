/* External Imports */
import { exec } from 'child_process'

/* Internal Imports */
import {
  VyperCompilationResult,
  VyperRawCompilationResult,
} from './interfaces/compilation.interface'

export const compile = async (
  path: string
): Promise<VyperCompilationResult> => {
  return new Promise<VyperCompilationResult>((resolve, reject) => {
    exec(`vyper ${path} -f combined_json`, (error, stdout) => {
      if (error) {
        reject(error)
      }
      const results = JSON.parse(stdout)
      const result: VyperRawCompilationResult = results[path]
      const parsed: VyperCompilationResult = {
        bytecode: result.bytecode,
        bytecodeRuntime: result.bytecode_runtime,
        abi: result.abi,
        sourceMap: result.source_map,
        methodIdentifiers: result.method_identifiers,
        version: results.version,
      }
      resolve(parsed)
    })
  })
}
