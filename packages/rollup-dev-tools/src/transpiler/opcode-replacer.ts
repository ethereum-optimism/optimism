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
  private readonly replacements: Map<
    EVMOpcode,
    EVMBytecode
  > = DefaultOpcodeReplacementsMap
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
    for (const [
      toReplace,
      bytecodeToReplaceWith,
    ] of opcodeReplacementBytecodes.entries()) {
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
      // for (let i = 0; i < bytecodeToReplaceWith.length; i++) {
      for (const opcodeAndBytesInReplacement of bytecodeToReplaceWith) {
        // ... replace execution manager plpaceholder
        if (
          opcodeAndBytesInReplacement.consumedBytes ===
          OpcodeReplacerImpl.EX_MGR_PLACEHOLDER
        ) {
          opcodeAndBytesInReplacement.consumedBytes = this.stateManagerAddressBuffer
        }
        // ...type check consumed bytes are the right length
        if (!isValidOpcodeAndBytes(opcodeAndBytesInReplacement)) {
          log.error(
            `Replacement config specified a ${
              opcodeAndBytesInReplacement.opcode.name
            } as an operation in the replacement bytecode for ${
              toReplace.name
            }, but the consumed bytes specified was ${bufToHexString(
              opcodeAndBytesInReplacement.consumedBytes
            )}--invalid length!`
          )
          throw new InvalidBytesConsumedError()
        }
      }
      // store the subbed and typechecked version in mapping
      this.replacements.set(toReplace, bytecodeToReplaceWith)
    }
  }

  public replaceIfNecessary(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
    if (!this.replacements.has(opcodeAndBytes.opcode)) {
      return [opcodeAndBytes]
    } else {
      return this.replacements.get(opcodeAndBytes.opcode)
    }
  }
}
