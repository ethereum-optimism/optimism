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
  if (codeAndBytes.consumedBytes !== undefined) {
    return (
      codeAndBytes.opcode.programBytesConsumed ===
      codeAndBytes.consumedBytes.length
    )
  } else {
    return codeAndBytes.opcode.programBytesConsumed === 0
  }
}
