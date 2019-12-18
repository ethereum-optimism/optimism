/* Internal Imports */
import { EVMOpcodeAndBytes } from '../'

/**
 * Validates that a given EVMOpcodeAndBytes has the right number of consumed bytes
 *
 * @param code the EVMOpcodeAndBytes which is to be verified.
 * @returns true if valid usage, false otherwise
 */
export const isValidOpcodeAndBytes = (
  codeAndBytes: EVMOpcodeAndBytes
): boolean => {
  return (
    codeAndBytes.opcode.programBytesConsumed ===
    codeAndBytes.consumedBytes.length
  )
}
