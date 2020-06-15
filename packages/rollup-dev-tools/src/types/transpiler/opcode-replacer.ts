/* External Imports */
import {
  EVMOpcodeAndBytes,
  EVMBytecode,
  EVMOpcode,
} from '@eth-optimism/rollup-core'

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export interface OpcodeReplacer {
  shouldSubstituteOpcodeForFunction(opcode: EVMOpcode): boolean
  getJUMPToOpcodeFunction(opcode: EVMOpcode): EVMBytecode
  getJUMPOnOpcodeFunctionReturn(opcode: EVMOpcode): EVMBytecode
  getOpcodeFunctionTable(opcodes: Set<EVMOpcode>): EVMBytecode
  populateOpcodeFunctionJUMPs(taggedBytecode: EVMBytecode): EVMBytecode
  getSubstituedFunctionFor(opcode: EVMOpcodeAndBytes): EVMBytecode
}
