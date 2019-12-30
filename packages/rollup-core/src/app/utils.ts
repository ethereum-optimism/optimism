/* Internal Imports */
import { EVMBytecode } from '../types'

/**
 * Takes EVMBytecode and serializes it into a single Buffer.
 *
 * @param bytecode The bytecode to serialize into a single Buffer.
 * @returns The resulting Buffer.
 */
export const bytecodeToBuffer = (bytecode: EVMBytecode): Buffer => {
  return Buffer.concat(
    bytecode.map((b) => {
      return !!b.consumedBytes
        ? Buffer.concat([b.opcode.code, b.consumedBytes])
        : b.opcode.code
    })
  )
}
