import { TranspilationResult } from './types'

/**
 * Interface defining the transpiler, which converts
 * EVM bytecode into Optimistic Rollup compatible EVM bytecode.
 */
export interface Transpiler {
  /**
   * Function to transpile input EVM bytecode according to configured Transpiler rules.
   * @param inputBytecode The bytecode to transpile
   * @returns The TranspilationResult, containing the list of errors if there are any
   * or the transpiled bytecode if successful.
   */
  transpile(inputBytecode: Buffer): TranspilationResult
}
