/* External Imports */
import { EVMOpcode } from '@pigi/rollup-core'

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export interface OpcodeReplacements {
  isOpcodeToReplace(opcode: EVMOpcode): boolean

  getOpcodeReplacement(opcode: EVMOpcode): Buffer
}