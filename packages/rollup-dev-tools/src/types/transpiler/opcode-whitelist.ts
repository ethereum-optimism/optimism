/* External Imports */
import { EVMOpcode } from '@pigi/rollup-core'

/**
 * Interface defining the available access operations for the OpCode Whitelist.
 */
export interface OpcodeWhitelist {
  isOpcodeWhitelisted(opcode: EVMOpcode): boolean

  isOpcodeWhitelistedByName(opcodeName: string): boolean

  isOpcodeWhitelistedByCodeBuffer(opcode: Buffer): boolean

  isOpcodeWhitelistedByCodeValue(opcode: number): boolean
}
