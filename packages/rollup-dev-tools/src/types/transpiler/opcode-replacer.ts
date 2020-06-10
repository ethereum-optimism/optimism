/* External Imports */
import { EVMOpcodeAndBytes, EVMBytecode, EVMOpcode } from '@eth-optimism/rollup-core'

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export interface OpcodeReplacer {
  shouldReplaceOpcode(opcode: EVMOpcode): boolean
  getJumpToReplacementInFooter(opcode: EVMOpcode): EVMBytecode
  replaceIfNecessary(opcode: EVMOpcodeAndBytes): EVMBytecode
}
