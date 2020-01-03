/* External Imports */
import { EVMOpcode, Opcode } from '@pigi/rollup-core'

/* Internal Imports */
import { OpcodeWhitelist } from '../../types/transpiler'

/**
 * Default and only intended implementation of OpcodeWhitelist.
 */
export class OpcodeWhitelistImpl implements OpcodeWhitelist {
  private readonly whitelist: Map<string, EVMOpcode>

  constructor(opcodes: EVMOpcode[]) {
    this.whitelist = new Map<string, EVMOpcode>(opcodes.map((x) => [x.name, x]))
  }

  public isOpcodeWhitelisted(opcode: EVMOpcode): boolean {
    return !!opcode && this.whitelist.has(opcode.name)
  }

  public isOpcodeWhitelistedByCodeBuffer(opcode: Buffer): boolean {
    const code: EVMOpcode = Opcode.parseByCode(opcode)
    return !!code && this.whitelist.has(code.name)
  }

  public isOpcodeWhitelistedByCodeValue(opcode: number): boolean {
    const code: EVMOpcode = Opcode.parseByNumber(opcode)
    return !!code && this.whitelist.has(code.name)
  }

  public isOpcodeWhitelistedByName(opcodeName: string): boolean {
    return !!opcodeName && this.whitelist.has(opcodeName)
  }
}
