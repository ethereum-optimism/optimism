/* External Imports */
import { EVMOpcodeAndBytes, EVMBytecode } from '@eth-optimism/rollup-core'

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export interface OpcodeReplacer {
  replaceIfNecessary(opcode: EVMOpcodeAndBytes): EVMBytecode
}
