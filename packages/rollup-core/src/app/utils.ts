/* Internal Imports */
import { EVMBytecode, Opcode } from '../types'
import { bufToHexString } from '@pigi/core-utils/build'

/**
 * Takes EVMBytecode and serializes it into a single Buffer.
 *
 * @param bytecode The bytecode to serialize into a single Buffer.
 * @returns The resulting Buffer.
 */
export const bytecodeToBuffer = (bytecode: EVMBytecode): Buffer => {
  return Buffer.concat(
    bytecode.map((b) => {
      return b.consumedBytes !== undefined
        ? Buffer.concat([b.opcode.code, b.consumedBytes])
        : b.opcode.code
    })
  )
}

/**
 * Parses the provided Buffer into EVMBytecode.
 * Note: If the Buffer is not valid bytecode, this will throw.
 *
 * @param buffer The buffer in question.
 * @returns The parsed EVMBytecode.
 */
export const bufferToBytecode = (buffer: Buffer): EVMBytecode => {
  const bytecode: EVMBytecode = []

  for (let pc = 0; pc < buffer.length; pc++) {
    const opcode = Opcode.parseByNumber(buffer[pc])
    const consumedBytes: Buffer =
      opcode.programBytesConsumed === 0
        ? undefined
        : buffer.slice(pc + 1, pc + 1 + opcode.programBytesConsumed)

    bytecode.push({
      opcode,
      consumedBytes,
    })

    pc += opcode.programBytesConsumed
  }
  return bytecode
}

/**
 * Gets the provided EVMBytecode as a printable string, where each line is an opcode and bytes.
 *
 * @param bytecode The EVMBytecode in question.
 * @returns The resulting string.
 */
export const formatBytecode = (bytecode: EVMBytecode): string => {
  return bytecode
    .map((x) => {
      if (x.consumedBytes === undefined) {
        return x.opcode.name
      }
      return `${x.opcode.name}: ${bufToHexString(x.consumedBytes)}`
    })
    .join('\n')
}
