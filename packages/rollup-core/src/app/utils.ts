/* Internal Imports */
import { EVMBytecode, EVMOpcodeAndBytes, Opcode } from '../types'
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
    if (!opcode) {
      bytecode.push({
        opcode: {
          name: `UNRECOGNIZED (${bufToHexString(Buffer.from([buffer[pc]]))})`,
          code: Buffer.from([buffer[pc]]),
          programBytesConsumed: 0,
        },
        consumedBytes: undefined,
      })
      continue
    }
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
    .map((x, index) => {
      let tagString: string = '(no tag)'
      if (!!x.tag) {
        tagString = `Metadata Tag: ${JSON.stringify(x.tag)}`
      }
      const pcAsString: string = padToLength(
        getPCOfEVMBytecodeIndex(index, bytecode),
        10
      )
      if (x.consumedBytes === undefined) {
        return `[PC ${pcAsString}] ${x.opcode.name} ${tagString}`
      }
      return `[PC ${pcAsString}] ${x.opcode.name}: ${bufToHexString(
        x.consumedBytes
      )} ${tagString}`
    })
    .join('\n')
}

const padToLength = (num: number, len: number): string => {
  const str = num.toString(16)
  return str.length >= len ? str : '0'.repeat(len - str.length) + str
}

/**
 * Gets the PC of the operation at a given index in some EVMBytecode.
 * In other words, it gives us the index of where a given element in some EVMBytecode would be in its raw Buffer form.
 *
 * @param indexOfEVMOpcodeAndBytes The index of an EVMOpcodeAndBytes element to find the PC of.
 * @param bytecode The EVMBytecode in question.
 * @returns The resulting index in raw bytes where the EVMOpcodeAndBytes begins.
 */
export const getPCOfEVMBytecodeIndex = (
  indexOfEVMOpcodeAndBytes: number,
  bytecode: EVMBytecode
): number => {
  let pc: number = 0
  for (let i = 0; i < indexOfEVMOpcodeAndBytes; i++) {
    const operation: EVMOpcodeAndBytes = bytecode[i]
    const totalBytesForOperation =
      operation.consumedBytes === undefined
        ? 1
        : 1 + operation.opcode.programBytesConsumed
    pc += totalBytesForOperation
  }
  return pc
}
