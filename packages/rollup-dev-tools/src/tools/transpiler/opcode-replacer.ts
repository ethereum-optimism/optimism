/* External Imports */
import {
  Opcode,
  EVMOpcode,
  EVMOpcodeAndBytes,
  EVMBytecode,
  isValidOpcodeAndBytes,
  Address,
} from '@pigi/rollup-core'
import {
  bufToHexString,
  remove0x,
  getLogger,
  isValidHexAddress,
} from '@pigi/core-utils'

/* Internal Imports */
import { OpcodeReplacer } from '../../types/transpiler'
import {
  InvalidAddressError,
  InvalidBytesConsumedError,
  UnsupportedOpcodeError,
} from '../../index'

const log = getLogger('transpiler:opcode-replacement')

export class OpcodeReplacerImpl implements OpcodeReplacer {
  public static EX_MGR_PLACEHOLDER: Buffer = Buffer.from(
    `{execution manager address placeholder}`
  )
  private readonly excutionManagerAddressBuffer: Buffer
  constructor(
    executionManagerAddress: Address,
    private readonly opcodeReplacementBytecodes: Map<EVMOpcode, EVMBytecode>
  ) {
    // check and store address
    if (!isValidHexAddress(executionManagerAddress)) {
      log.error(
        `Opcode replacer received ${executionManagerAddress} for the execution manager address.  Not a valid hex string address!`
      )
      throw new InvalidAddressError()
    } else {
      this.excutionManagerAddressBuffer = Buffer.from(
        remove0x(executionManagerAddress),
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
      for (const opcodeAndBytesInReplacement of bytecodeToReplaceWith) {
        // ... replace execution manager plpaceholder
        if (
          opcodeAndBytesInReplacement.consumedBytes ===
          OpcodeReplacerImpl.EX_MGR_PLACEHOLDER
        ) {
          opcodeAndBytesInReplacement.consumedBytes = this.excutionManagerAddressBuffer
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
            )}--invalid length! (length ${
              opcodeAndBytesInReplacement.consumedBytes.length
            })`
          )
          throw new InvalidBytesConsumedError()
        }
      }
    }
  }

  /**
   * Gets the specified replacement bytecode for a given EVM opcode and bytes
   * @param opcodeAndBytes EVM opcode and consumed bytes which is supposed to be replaced.
   *
   * @returns The EVMBytecode we have decided to replace opcodeAndBytes with.
   */
  public replaceIfNecessary(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
    if (!this.opcodeReplacementBytecodes.has(opcodeAndBytes.opcode)) {
      return [opcodeAndBytes]
    } else {
      return this.opcodeReplacementBytecodes.get(opcodeAndBytes.opcode)
    }
  }
}
