/* External Imports */
import {
  Opcode as Ops,
  EVMOpcode,
  EVMOpcodeAndBytes,
  EVMBytecode,
  isValidOpcodeAndBytes,
  Address,
  Opcode,
} from '@pigi/rollup-core'

/* Internal Imports */
import { OpcodeReplacer } from '../types/transpiler'
import {
  InvalidAddressError,
  InvalidBytesConsumedError,
  UnsupportedOpcodeError,
} from '../'
import {
  hexStrToBuf,
  bufToHexString,
  remove0x,
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

const DefaultOpcodeReplacementsMap: Map<EVMOpcode, EVMBytecode> = new Map<
  EVMOpcode,
  EVMBytecode
>()
  .set(Ops.ADDMOD, [
    { opcode: Ops.AND, consumedBytes: undefined },
    { opcode: undefined, consumedBytes: undefined },
  ])
  .set(Ops.BYTE, [
    { opcode: undefined, consumedBytes: undefined },
    { opcode: undefined, consumedBytes: undefined },
  ])

export class OpcodeReplacerImpl implements OpcodeReplacer {
  public static EX_MGR_PLACEHOLDER: Buffer = Buffer.from(
    `{execution manager address placeholder}`
  )
  private readonly stateManagerAddressBuffer: Buffer
  private readonly replacements: Map<EVMOpcode, EVMBytecode> = new Map<
    EVMOpcode,
    EVMBytecode
  >()
  constructor(
    stateManagerAddress: Address,
    opcodeReplacementBytecodes: Map<
      EVMOpcode,
      EVMBytecode
    > = DefaultOpcodeReplacementsMap
  ) {
    // check and store address
    if (!isValidHexAddress(stateManagerAddress)) {
      log.error(
        `Opcode replacer recieved ${stateManagerAddress} for the state manager address.  Not a valid hex string address!`
      )
      throw new InvalidAddressError()
    } else {
      this.stateManagerAddressBuffer = Buffer.from(
        remove0x(stateManagerAddress),
        'hex'
      )
    }
    for (const entry of opcodeReplacementBytecodes.entries()) {
      const toReplace: EVMOpcode = entry[0]
      const bytecodeToRelpaceWith: EVMBytecode = entry[1]
      // Make sure we're not attempting to overwrite PUSHN, not yet supported
      if (toReplace.programBytesConsumed > 0) {
        log.error(
          `Transpilation currently does not support opcodes which consume bytes, but config specified a replacement for ${JSON.stringify(
            toReplace
          )}.`
        )
        throw new UnsupportedOpcodeError()
      }
      // for each operation in the replacement bytecode for this toReplace...
      for (let i = 0; i < bytecodeToRelpaceWith.length; i++) {
        const opcodeAndBytesInReplacement: EVMOpcodeAndBytes =
          bytecodeToRelpaceWith[i]
        // ... replace execution manager plpaceholder
        if (
          opcodeAndBytesInReplacement.consumedBytes ===
          OpcodeReplacerImpl.EX_MGR_PLACEHOLDER
        ) {
          bytecodeToRelpaceWith[
            i
          ].consumedBytes = this.stateManagerAddressBuffer
        }
        // ...type check consumed bytes are the right length
        if (!isValidOpcodeAndBytes(opcodeAndBytesInReplacement)) {
          log.error(
            `Replacement config specified a ${
              opcodeAndBytesInReplacement.opcode.name
            } as the ${i}th operation in the replacement bytecode for ${
              toReplace.name
            }, but the consumed bytes specified was ${bufToHexString(
              opcodeAndBytesInReplacement.consumedBytes
            )}--invalid length!`
          )
          throw new InvalidBytesConsumedError()
        }
      }
      // store the subbed and typechecked version in mapping
      this.replacements.set(toReplace, bytecodeToRelpaceWith)
    }
  }

  public replaceIfNecessary(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
    if (!this.replacements.has(opcodeAndBytes.opcode)) {
      return [opcodeAndBytes]
    } else {
      return this.replacements.get(opcodeAndBytes.opcode)
    }
  }

  // public getOpcodeReplacement(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
  //   log.debug(
  //     `Recieved a replacement request for ${JSON.stringify(opcodeAndBytes)}.`
  //   )
  //   if (opcodeAndBytes.consumedBytes !== undefined) {
  //     if (opcodeAndBytes.consumedBytes.length > 0) {
  //       log.debug(
  //         `Transpilation currently does not support opcodes which consume bytes.  Transpiler requested a replacement for ${JSON.stringify(
  //           opcodeAndBytes
  //         )}.  Thus, not replacing, just returning...`
  //       )
  //       return [opcodeAndBytes]
  //     }
  //   }
  //   const opcodeToReplace: EVMOpcode = opcodeAndBytes.opcode
  //   const cfgReplacmentArray: string[] = this.replacementConfig[
  //     opcodeToReplace.name
  //   ]
  //   log.debug(
  //     `The replacement string for opcode [${
  //       opcodeToReplace.name
  //     }] is: ${JSON.stringify(cfgReplacmentArray)}.  Parsing...`
  //   )

  //   // init replacement bytecode
  //   const replacementBytecode: EVMBytecode = []

  //   for (let i = 0; i < cfgReplacmentArray.length; i++) {
  //     if (cfgReplacmentArray[i] === PUSH_STATE_MGR_ADDR) {
  //       const stateManagerPush: EVMOpcodeAndBytes = {
  //         opcode: Ops.PUSH20,
  //         consumedBytes: hexStrToBuf(this.stateManagerAddress),
  //       }
  //       log.debug(
  //         `Found a request to push the staet manager address at index ${i} in the replacement array.  Putting in as ${JSON.stringify(
  //           stateManagerPush
  //         )}`
  //       )
  //       replacementBytecode.push(stateManagerPush)
  //       continue
  //     }

  //     const opInReplacement: EVMOpcode = Ops.parseByName(cfgReplacmentArray[i])
  //     if (opInReplacement === undefined) {
  //       log.error(
  //         `Opcode replacement config JSON specified: [${cfgReplacmentArray[i]}] at index ${i}, which could not be parsed into an EVM Opcode to return.`
  //       )
  //       // throw new OpcodeParseError()
  //     }
  //     log.debug(
  //       `Parsing the ${i}th opcode in the replacement for ${opcodeToReplace.name}, its name is: ${opInReplacement.name}.  Adding to replacement bytecode.`
  //     )

  //     let consumedValueBuffer: Buffer
  //     // sanitize/typecheck PUSHes
  //     const bytesToConsume: number = opInReplacement.programBytesConsumed
  //     if (bytesToConsume > 0) {
  //       log.debug(
  //         `Parsed the ${i}th bytecode replacement element for ${opcodeToReplace.name} to be ${opInReplacement.name}-- which is expected to consume ${bytesToConsume} bytes.`
  //       )
  //       consumedValueBuffer = hexStrToBuf(cfgReplacmentArray[i + 1])
  //       // skip over the consumed bytees for next iteration of parsing
  //       i++
  //     }
  //     const parsedReplacementOpcodeAndBytes: EVMOpcodeAndBytes = {
  //       opcode: opInReplacement,
  //       consumedBytes: consumedValueBuffer,
  //     }

  //     if (!isValidOpcodeAndBytes(parsedReplacementOpcodeAndBytes)) {
  //       log.error(
  //         `Replacement config specified a ${
  //           opInReplacement.name
  //         } as the ${i}th element, but the ${i +
  //           1}th element was ${bufToHexString(
  //           consumedValueBuffer
  //         )}--invalid length!`
  //       )
  //       throw new InvalidBytesConsumedError()
  //     } else {
  //       log.debug(
  //         `The proceeding hex string was found to be the right length for this [${opInReplacement.name}].  Continuing...`
  //       )
  //     }
  //     replacementBytecode.push(parsedReplacementOpcodeAndBytes)
  //   }
  //   log.info(
  //     `Replacement Bytecode for opcode [${
  //       opcodeToReplace.name
  //     }] was parsed to: ${JSON.stringify(replacementBytecode)}.  Returning...`
  //   )
  //   return replacementBytecode
  // }
}
