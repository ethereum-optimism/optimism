import { TranspilationResult } from './types'

/**
 * Interface defining the transpiler, which converts
 * EVM bytecode into Optimistic Rollup compatible EVM bytecode.
 */
export interface Transpiler {
  /**
   * Function to transpile input bytecode & deployedBytecode according to configured Transpiler rules.
   * Note: The resulting bytecode will work if used in CREATE or CREATE2
   *
   * @param bytecode The bytecode (initcode + deployedBytecode) to transpile.
   * @param deployedBytecode The deployedBytecode to transpile.
   * @param originalDeployedBytecodeSize The size of the original initcode with auxdata if different than deployedBytecode length.
   * @returns The TranspilationResult, containing the list of errors if there are any
   * or the transpiled bytecode if successful.
   */
  transpile(
    bytecode: Buffer,
    deployedBytecode: Buffer,
    originalDeployedBytecodeSize?: number
  ): TranspilationResult

  /**
   * Function to transpile input rawBytecode according to configured Transpiler rules.
   * Note: This function will work for raw bytecode but the resulting bytecode will fail if used in CREATE or CREATE2
   *
   * @param rawBytecode The raw bytecode to transpile
   * @returns The TranspilationResult, containing the list of errors if there are any
   * or the transpiled bytecode if successful.
   */
  transpileRawBytecode(rawBytecode: Buffer): TranspilationResult
}
