/* External Imports */
import { EVMOpcode, EVMBytecode } from '@pigi/rollup-core'

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export interface OpcodeReplacer {
  isOpcodeToReplace(opcode: EVMOpcode): boolean

  getOpcodeReplacement(opcode: EVMOpcode): EVMBytecode
}
