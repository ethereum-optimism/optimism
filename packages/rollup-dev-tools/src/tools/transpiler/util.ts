import { TranspilationError } from '../../types/transpiler'

export const BIG_ENOUGH_GAS_LIMIT: number = 100000000

/**
 * Util function to create TranspilationErrors.
 *
 * @param index The index of the byte in the input bytecode where the error originates.
 * @param error The TranspilationError error type.
 * @param message The error message.
 * @returns The constructed TranspilationError
 */
export const createError = (
  index: number,
  error: number,
  message: string
): TranspilationError => {
  return {
    index,
    error,
    message,
  }
}
