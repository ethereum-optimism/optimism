/* External Imports */
import { EVMOpcodeAndBytes, EVMBytecode } from '@pigi/rollup-core'

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export interface OpcodeReplacer {
  getOpcodeReplacement(opcode: EVMOpcodeAndBytes): EVMBytecode
}
