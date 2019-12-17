/* External Imports */
import { EVMOpcode, Opcode, EVMBytecode, Address } from '@pigi/rollup-core'

/* Internal Imports */
import { OpcodeReplacements } from '../types/transpiler'
import { hexStrToBuf, getLogger } from '@pigi/core-utils'

const log = getLogger('transpiler:opcode-replacement')

// placeholder command for pushing state manager address onto stack
const PUSH_STATE_MGR_ADDR = 'PUSH_STATE_MGR_ADDR'
// the desired replacments themselves -- strings for opcodes, hex strings for pushable bytes
const OpcodeReplacementsJSON = {
  PUSH1: ['PUSH1', '0x00', PUSH_STATE_MGR_ADDR],
  PUSH2: ['ADD', 'PUSH2', '0x0000'],
}

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export class OpcodeReplacementsImpl implements OpcodeReplacements {
  private readonly replacedOpcodes: EVMOpcode[] = []
  private readonly opcodeReplacementBytecodes: Map<
    EVMOpcode,
    EVMBytecode
  > = new Map<EVMOpcode, EVMBytecode>()

  constructor(stateManagerAddress: Address) {
    log.debug(
      `Parsing the following opcode replacement JSON string config: ${JSON.stringify(
        OpcodeReplacementsJSON
      )}`
    )
    for (const opcodeToReplaceStr in OpcodeReplacementsJSON) { // tslint:disable-line
      const opcodeToReplace: EVMOpcode = Opcode.parseByName(opcodeToReplaceStr)
      log.debug(
        `Parsing bytecode replacement for opcode [${opcodeToReplace.name}].`
      )

      // get the specified replacement array from the JSON config
      const replacementArray: string[] =
        OpcodeReplacementsJSON[opcodeToReplaceStr]
      log.info(
        `The JSON specified a replacement array as: ${JSON.stringify(
          replacementArray
        )}`
      )

      // We want to replace all PUSH_STATE_MGR_ADDR with PUSH20 + the configured address.  Currently only works with max P_S_M_A per replacement.
      // Find where it is, if anywhere
      const indexToReplaceAddressPush = replacementArray.indexOf(
        PUSH_STATE_MGR_ADDR
      )
      if (indexToReplaceAddressPush >= 0) {
        log.debug(
          `Found a PUSH_STATE_MGR_ADDR at index ${indexToReplaceAddressPush}.  Splicing in ['PUSH20', '${stateManagerAddress.toString()}'].`
        )
        // replace free var with PUSH20, 20-byte State Mgr Address
        replacementArray.splice(
          indexToReplaceAddressPush,
          1,
          'PUSH20',
          stateManagerAddress
        )
      }

      // now convert everything to bytcode, making sure PUSHN is always followed by length-N buffer
      const replacementAsBytecode: EVMBytecode = []
      for (let i = 0; i < replacementArray.length; i++) {
        const op: EVMOpcode = Opcode.parseByName(replacementArray[i])
        if (op === undefined) {
          log.error(
            `Opcode replacement config JSON specified: [${replacementArray[i]}], which could not be parsed into an EVM Opcode.`
          )
          process.exit(1)
        }
        log.debug(
          `Parsing the ${i}th opcode in the replacement for ${opcodeToReplaceStr}, this opcode is: ${op.name}`
        )
        replacementAsBytecode[i] = op

        // sanitize/typecheck PUSHes
        const bytesToConsume: number = op.programBytesConsumed
        if (bytesToConsume > 0) {
          log.debug(
            `Parsed the ${i}th bytcode replacement element for ${opcodeToReplaceStr} to be ${op.name}-- which is expected to consume ${bytesToConsume}.`
          )
          const consumedValueBuffer: Buffer = hexStrToBuf(
            replacementArray[i + 1]
          )
          if (consumedValueBuffer === undefined) {
            log.error(
              `Final opcode in replacement array for ${opcodeToReplaceStr} was ${op.name}, but was not proceeded by any bytes to consume.`
            )
            process.exit(1)
          }

          if (consumedValueBuffer.length !== bytesToConsume) {
            log.error(
              `The hex sring following the PUSH operation was 0x[${consumedValueBuffer.toString(
                'hex'
              )}], but was expecting ${bytesToConsume} bytes to consume.`
            )
          }
          log.debug(
            `The proceeding hex string was found to be the right length for this [${op.name}].  Continuing...`
          )
          replacementAsBytecode[i + 1] = consumedValueBuffer
          i++
        }
        log.info(
          `Storing replacement Bytecode for [${
            opcodeToReplace.name
          }] as: ${JSON.stringify(replacementAsBytecode)}.`
        )
        // Store that we are replacing this one
        this.replacedOpcodes.push(opcodeToReplace)
        // Store its replacement
        this.opcodeReplacementBytecodes.set(
          opcodeToReplace,
          replacementAsBytecode
        )
      }
    }
  }

  public isOpcodeToReplace(opcode: EVMOpcode): boolean {
    return this.replacedOpcodes.includes(opcode)
  }

  public getOpcodeReplacement(opcode: EVMOpcode): EVMBytecode {
    if (this.isOpcodeToReplace(opcode)) {
        return this.opcodeReplacementBytecodes.get(opcode)
    } else {
        return [opcode]
    }
  }
}
