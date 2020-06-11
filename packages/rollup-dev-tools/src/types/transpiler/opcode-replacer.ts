/* External Imports */
import { EVMOpcodeAndBytes, EVMBytecode, EVMOpcode } from '@eth-optimism/rollup-core'

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export interface OpcodeReplacer {
  shouldReplaceOpcode(opcode: EVMOpcode): boolean
  getJUMPToReplacementInFooter(opcode: EVMOpcode): EVMBytecode
  getOpcodeReplacementFooter(opcodes: Set<EVMOpcode>): EVMBytecode
  fixOpcodeReplacementJUMPs(taggedBytecode: EVMBytecode): EVMBytecode
  replaceIfNecessary(opcode: EVMOpcodeAndBytes): EVMBytecode
}
