/* External Imports */
import { EVMOpcode, Opcode } from '@eth-optimism/rollup-core'

/* Internal Imports */
import { OpcodeWhitelist } from '../../types/transpiler'

/**
 * Default and only intended implementation of OpcodeWhitelist.
 */
export class OpcodeWhitelistImpl implements OpcodeWhitelist {
  private readonly whitelist: Map<string, EVMOpcode>

  constructor(opcodes: EVMOpcode[] = defaultWhitelist) {
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

const defaultWhitelist: EVMOpcode[] = [
  Opcode.ADD,
  Opcode.ADDMOD,
  Opcode.ADDRESS,
  Opcode.AND,
  Opcode.BYTE,
  Opcode.CALL,
  Opcode.CALLDATACOPY,
  Opcode.CALLDATALOAD,
  Opcode.CALLDATASIZE,
  Opcode.CALLER,
  Opcode.CALLVALUE,
  Opcode.CODECOPY,
  Opcode.CODESIZE,
  Opcode.CREATE,
  Opcode.CREATE2,
  Opcode.DELEGATECALL,
  Opcode.DIV,
  Opcode.DUP1,
  Opcode.DUP10,
  Opcode.DUP11,
  Opcode.DUP12,
  Opcode.DUP13,
  Opcode.DUP14,
  Opcode.DUP15,
  Opcode.DUP16,
  Opcode.DUP2,
  Opcode.DUP3,
  Opcode.DUP4,
  Opcode.DUP5,
  Opcode.DUP6,
  Opcode.DUP7,
  Opcode.DUP8,
  Opcode.DUP9,
  Opcode.EQ,
  Opcode.EXP,
  Opcode.EXTCODECOPY,
  Opcode.EXTCODESIZE,
  Opcode.EXTCODEHASH,
  Opcode.GAS,
  Opcode.GT,
  Opcode.INVALID,
  Opcode.ISZERO,
  Opcode.JUMP,
  Opcode.JUMPDEST,
  Opcode.JUMPI,
  Opcode.LOG0,
  Opcode.LOG1,
  Opcode.LOG2,
  Opcode.LOG3,
  Opcode.LOG4,
  Opcode.LT,
  Opcode.MLOAD,
  Opcode.MOD,
  Opcode.MSIZE,
  Opcode.MSTORE,
  Opcode.MSTORE8,
  Opcode.MUL,
  Opcode.MULMOD,
  Opcode.NOT,
  Opcode.OR,
  Opcode.ORIGIN,
  Opcode.PC,
  Opcode.POP,
  Opcode.PUSH1,
  Opcode.PUSH10,
  Opcode.PUSH11,
  Opcode.PUSH12,
  Opcode.PUSH13,
  Opcode.PUSH14,
  Opcode.PUSH15,
  Opcode.PUSH16,
  Opcode.PUSH17,
  Opcode.PUSH18,
  Opcode.PUSH19,
  Opcode.PUSH2,
  Opcode.PUSH20,
  Opcode.PUSH21,
  Opcode.PUSH22,
  Opcode.PUSH23,
  Opcode.PUSH24,
  Opcode.PUSH25,
  Opcode.PUSH26,
  Opcode.PUSH27,
  Opcode.PUSH28,
  Opcode.PUSH29,
  Opcode.PUSH3,
  Opcode.PUSH30,
  Opcode.PUSH31,
  Opcode.PUSH32,
  Opcode.PUSH4,
  Opcode.PUSH5,
  Opcode.PUSH6,
  Opcode.PUSH7,
  Opcode.PUSH8,
  Opcode.PUSH9,
  Opcode.RETURN,
  Opcode.RETURNDATACOPY,
  Opcode.RETURNDATASIZE,
  Opcode.REVERT,
  Opcode.SAR,
  Opcode.SDIV,
  Opcode.SGT,
  Opcode.SHA3,
  Opcode.SHL,
  Opcode.SHR,
  Opcode.SIGNEXTEND,
  Opcode.SLT,
  Opcode.SMOD,
  Opcode.SLOAD,
  Opcode.SSTORE,
  Opcode.STATICCALL,
  Opcode.STOP,
  Opcode.SUB,
  Opcode.SWAP1,
  Opcode.SWAP10,
  Opcode.SWAP11,
  Opcode.SWAP12,
  Opcode.SWAP13,
  Opcode.SWAP14,
  Opcode.SWAP15,
  Opcode.SWAP16,
  Opcode.SWAP2,
  Opcode.SWAP3,
  Opcode.SWAP4,
  Opcode.SWAP5,
  Opcode.SWAP6,
  Opcode.SWAP7,
  Opcode.SWAP8,
  Opcode.SWAP9,
  Opcode.TIMESTAMP,
  Opcode.XOR,
]
