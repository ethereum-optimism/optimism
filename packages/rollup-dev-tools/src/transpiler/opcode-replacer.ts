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
  InvalidAddressError,
  OpcodeParseError,
  InvalidBytesConsumedError,
} from '../'
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

const DefaultOpcodeReplacements = {
  PUSH1: ['PUSH1', '0x00', PUSH_STATE_MGR_ADDR],
  PUSH2: ['ADD', 'PUSH2', '0x0000'],
}

export class OpcodeReplacerImpl implements OpcodeReplacer {
  private readonly opcodeReplacementBytecodes: Map<
    EVMOpcode,
    EVMBytecode
  > = new Map<EVMOpcode, EVMBytecode>()

  constructor(
    private readonly stateManagerAddress: Address,
    private replacementConfig = DefaultOpcodeReplacements as any
  ) {
    if (!isValidHexAddress(stateManagerAddress)) {
      log.error(
        `Opcode replacer recieved ${stateManagerAddress} for the state manager address.  Not a valid hex string address!`
      )
      throw new InvalidAddressError()
    }
  }

  public getOpcodeReplacement(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
    log.debug(
      `Recieved a replacement request for ${JSON.stringify(opcodeAndBytes)}.`
    )
    if (opcodeAndBytes.consumedBytes !== undefined) {
      if (opcodeAndBytes.consumedBytes.length > 0) {
        log.debug(
          `Transpilation currently does not support opcodes which consume bytes.  Transpiler requested a replacement for ${JSON.stringify(
            opcodeAndBytes
          )}.  Thus, not replacing, just returning...`
        )
        return [opcodeAndBytes]
      }
    }
    const opcodeToReplace: EVMOpcode = opcodeAndBytes.opcode
    const cfgReplacmentArray: string[] = this.replacementConfig[
      opcodeToReplace.name
    ]
    log.debug(
      `The replacement string for opcode [${
        opcodeToReplace.name
      }] is: ${JSON.stringify(cfgReplacmentArray)}.  Parsing...`
    )

    // init replacement bytecode
    const replacementBytecode: EVMBytecode = []

    for (let i = 0; i < cfgReplacmentArray.length; i++) {
      if (cfgReplacmentArray[i] === PUSH_STATE_MGR_ADDR) {
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

      const opInReplacement: EVMOpcode = Ops.parseByName(cfgReplacmentArray[i])
      if (opInReplacement === undefined) {
        log.error(
          `Opcode replacement config JSON specified: [${cfgReplacmentArray[i]}] at index ${i}, which could not be parsed into an EVM Opcode to return.`
        )
        throw new OpcodeParseError()
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
        consumedValueBuffer = hexStrToBuf(cfgReplacmentArray[i + 1])
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
        throw new InvalidBytesConsumedError()
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
