/* External Imports */
import {
  Opcode as Ops,
  EVMOpcode,
  EVMOpcodeAndBytes,
  EVMBytecode,
  isValidOpcodeAndBytes,
  Address,
} from '@pigi/rollup-core'

/* Internal Imports */
import { OpcodeReplacer } from '../types/transpiler'
import {
  hexStrToBuf,
  bufToHexString,
  getLogger,
  isValidHexAddress,
} from '@pigi/core-utils'

const log = getLogger('transpiler:opcode-replacement')

// placeholder command for pushing state manager address onto stack
const PUSH_STATE_MGR_ADDR = 'PUSH_STATE_MGR_ADDR'
// the desired replacments themselves -- strings for opcodes, hex strings for pushable bytes
const OpcodeReplacementsJSON = {
  PUSH1: ['PUSH1', '0x00', PUSH_STATE_MGR_ADDR],
  PUSH2: ['ADD', 'PUSH2', '0x0000'],
}

const DefaultOpcodeReplacements = {
  PUSH1: ['PUSH1', '0x00', PUSH_STATE_MGR_ADDR],
  PUSH2: ['ADD', 'PUSH2', '0x0000'],
}

/**
 * Interface defining the set of transpiled opcodes, and what bytecode to replace with.
 */
export class OpcodeReplacerImpl implements OpcodeReplacer {
  private readonly opcodeReplacementBytecodes: Map<
    EVMOpcode,
    EVMBytecode
  > = new Map<EVMOpcode, EVMBytecode>()

  constructor(private readonly stateManagerAddress: Address) {
    if (!isValidHexAddress(stateManagerAddress)) {
      log.error(
        `Opcode replacer recieved ${stateManagerAddress} for the state manager address.  Not a valid hex string address!`
      )
    }
  }
  //   log.debug(
  //     `Opcode replacer will be using the following opcode replacement JSON config: ${JSON.stringify(
  //       OpcodeReplacementsJSON
  //     )}`
  //   )
  //   for (const opcodeToReplaceStr in OpcodeReplacementsJSON) { // tslint:disable-line
  //     const opcodeToReplace: EVMOpcode = Opcode.parseByName(opcodeToReplaceStr)
  //     log.debug(
  //       `Parsing bytecode replacement for opcode [${opcodeToReplace.name}].`
  //     )

  //     // get the specified replacement array from the JSON config
  //     const replacementArray: string[] =
  //       OpcodeReplacementsJSON[opcodeToReplaceStr]
  //     log.info(
  //       `The JSON specified a replacement array as: ${JSON.stringify(
  //         replacementArray
  //       )}`
  //     )

  //     // We want to replace all PUSH_STATE_MGR_ADDR with PUSH20 + the configured address.  Currently only works with max P_S_M_A per replacement.
  //     // Find where it is, if anywhere
  //     const indexToReplaceAddressPush = replacementArray.indexOf(
  //       PUSH_STATE_MGR_ADDR
  //     )
  //     if (indexToReplaceAddressPush >= 0) {
  //       log.debug(
  //         `Found a PUSH_STATE_MGR_ADDR at index ${indexToReplaceAddressPush}.  Splicing in ['PUSH20', '${stateManagerAddress.toString()}'].`
  //       )
  //       // replace free var with PUSH20, 20-byte State Mgr Address
  //       replacementArray.splice(
  //         indexToReplaceAddressPush,
  //         1,
  //         'PUSH20',
  //         stateManagerAddress
  //       )
  //     }

  //     // now convert everything to bytcode, making sure PUSHN is always followed by length-N buffer
  //     const replacementAsBytecode: EVMBytecode = []
  //     for (let i = 0; i < replacementArray.length; i++) {
  //       const op: EVMOpcode = Opcode.parseByName(replacementArray[i])
  //       if (op === undefined) {
  //         log.error(
  //           `Opcode replacement config JSON specified: [${replacementArray[i]}], which could not be parsed into an EVM Opcode.`
  //         )
  //         process.exit(1)
  //       }
  //       log.debug(
  //         `Parsing the ${i}th opcode in the replacement for ${opcodeToReplaceStr}, this opcode is: ${op.name}`
  //       )
  //       replacementAsBytecode[i] = op

  //       // sanitize/typecheck PUSHes
  //       const bytesToConsume: number = op.programBytesConsumed
  //       if (bytesToConsume > 0) {
  //         log.debug(
  //           `Parsed the ${i}th bytcode replacement element for ${opcodeToReplaceStr} to be ${op.name}-- which is expected to consume ${bytesToConsume}.`
  //         )
  //         const consumedValueBuffer: Buffer = hexStrToBuf(
  //           replacementArray[i + 1]
  //         )
  //         if (consumedValueBuffer === undefined) {
  //           log.error(
  //             `Final opcode in replacement array for ${opcodeToReplaceStr} was ${op.name}, but was not proceeded by any bytes to consume.`
  //           )
  //           process.exit(1)
  //         }

  //         if (consumedValueBuffer.length !== bytesToConsume) {
  //           log.error(
  //             `The hex sring following the PUSH operation was 0x[${consumedValueBuffer.toString(
  //               'hex'
  //             )}], but was expecting ${bytesToConsume} bytes to consume.`
  //           )
  //         }
  //         log.debug(
  //           `The proceeding hex string was found to be the right length for this [${op.name}].  Continuing...`
  //         )
  //         replacementAsBytecode[i + 1] = consumedValueBuffer
  //         i++
  //       }
  //       log.info(
  //         `Storing replacement Bytecode for [${
  //           opcodeToReplace.name
  //         }] as: ${JSON.stringify(replacementAsBytecode)}.`
  //       )
  //       // Store that we are replacing this one
  //       this.replacedOpcodes.push(opcodeToReplace)
  //       // Store its replacement
  //       this.opcodeReplacementBytecodes.set(
  //         opcodeToReplace,
  //         replacementAsBytecode
  //       )
  //     }
  //   }
  // }

  public getOpcodeReplacement(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
    log.debug(
      `Recieved a replacement request for ${JSON.stringify(opcodeAndBytes)}.`
    )
    if (opcodeAndBytes.consumedBytes.length > 0) {
      log.debug(
        `Transpilation currently does not support opcodes which consume bytes.  Transpiler requested a replacement for ${JSON.stringify(
          opcodeAndBytes
        )}.  Not replacing, just returning...`
      )
      return [opcodeAndBytes]
    }
    const opcodeToReplace: EVMOpcode = opcodeAndBytes.opcode
    const replacementConfig: string[] =
      DefaultOpcodeReplacements[opcodeToReplace.name]
    log.debug(
      `The replacement string for opcode [${
        opcodeToReplace.name
      }] is: ${JSON.stringify(replacementConfig)}.  Parsing...`
    )

    // init replacement bytecode
    const replacementBytecode: EVMBytecode = []

    // We want to replace all PUSH_STATE_MGR_ADDR with PUSH20 + the configured address.  Currently only works with max P_S_M_A per replacement.
    // Find where it is, if anywhere
    // const indexToReplaceAddressPush = replacementConfig.indexOf(
    //   PUSH_STATE_MGR_ADDR
    // )
    // if (indexToReplaceAddressPush >= 0) {
    //   log.debug(
    //     `Found a PUSH_STATE_MGR_ADDR at index ${indexToReplaceAddressPush}.  Splicing in a PUSH20 with SM address '${stateManagerAddress.toString()}'].`
    //   )
    //   // replace free var with PUSH20, 20-byte State Mgr Address
    //   replacementBytecode.push({
    //     opcode: Ops.PUSH20,
    //     consumedBytes: hexStrToBuf(this.stateManagerAddress)
    //   })
    //   continue
    // }

    for (let i = 0; i < replacementConfig.length; i++) {
      if (replacementConfig[i] === PUSH_STATE_MGR_ADDR) {
        const stateManagerPush: EVMOpcodeAndBytes = {
          opcode: Ops.PUSH20,
          consumedBytes: hexStrToBuf(this.stateManagerAddress),
        }
        log.debug(
          `Found a request to push the staet manager address at index ${i} in the replacement array.  Putting in as ${JSON.stringify(
            stateManagerPush
          )}`
        )
        replacementBytecode.push(stateManagerPush)
        continue
      }

      const opInReplacement: EVMOpcode = Ops.parseByName(replacementConfig[i])
      if (opInReplacement === undefined) {
        log.error(
          `Opcode replacement config JSON specified: [${replacementConfig[i]}] at index ${i}, which could not be parsed into an EVM Opcode to return.`
        )
        process.exit(1)
      }
      log.debug(
        `Parsing the ${i}th opcode in the replacement for ${opcodeToReplace.name}, its name is: ${opInReplacement.name}.  Adding to replacement bytecode.`
      )

      let consumedValueBuffer: Buffer
      // sanitize/typecheck PUSHes
      const bytesToConsume: number = opInReplacement.programBytesConsumed
      if (bytesToConsume > 0) {
        log.debug(
          `Parsed the ${i}th bytecode replacement element for ${opcodeToReplace.name} to be ${opInReplacement.name}-- which is expected to consume ${bytesToConsume} bytes.`
        )
        consumedValueBuffer = hexStrToBuf(replacementConfig[i + 1])
        // skip over the consumed bytees for next iteration of parsing
        i++
      }
      const parsedReplacementOpcodeAndBytes: EVMOpcodeAndBytes = {
        opcode: opInReplacement,
        consumedBytes: consumedValueBuffer,
      }

      if (!isValidOpcodeAndBytes(parsedReplacementOpcodeAndBytes)) {
        log.error(
          `Replacement config specified a ${
            opInReplacement.name
          } as the ${i}th element, but the ${i +
            1}th element was ${bufToHexString(
            consumedValueBuffer
          )}--invalid length!`
        )
      } else {
        log.debug(
          `The proceeding hex string was found to be the right length for this [${opInReplacement.name}].  Continuing...`
        )
      }
      replacementBytecode.push(parsedReplacementOpcodeAndBytes)
    }
    log.info(
      `Replacement Bytecode for opcode [${
        opcodeToReplace.name
      }] was parsed to: ${JSON.stringify(replacementBytecode)}.  Returning...`
    )
    return replacementBytecode
  }
}
