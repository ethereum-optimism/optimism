import { VyperAbiMethod } from './abi.interface'
import { VyperSourceMap } from './source-map.interface'

export interface VyperCompilationResult {
  bytecode: string
  bytecodeRuntime: string
  abi: VyperAbiMethod[]
  sourceMap: VyperSourceMap
  methodIdentifiers: { [key: string]: string }
  version: string
}

export interface VyperRawCompilationResult {
  bytecode: string
  bytecode_runtime: string
  abi: VyperAbiMethod[]
  source_map: VyperSourceMap
  method_identifiers: { [key: string]: string }
}
