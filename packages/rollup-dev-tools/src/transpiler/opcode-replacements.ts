/* External Imports */
import { EVMOpcode } from '@pigi/rollup-core'
import { remove0x } from '@pigi/core-utils';

/* Internal Imports */
import { OpcodeReplacements } from '../types/transpiler'

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export class OpcodeReplacementsImpl implements OpcodeReplacements {
    private readonly whitelist: Map<string, EVMOpcode>
  
    constructor(opcodes: EVMOpcode[]) {
      this.whitelist = new Map<string, EVMOpcode>(opcodes.map((x) => [x.name, x]))
    }

    public isOpcodeToReplace(opcode: EVMOpcode): boolean {
        return
    }

    public getOpcodeReplacement(opcode: EVMOpcode): Buffer {
        return
    }
}