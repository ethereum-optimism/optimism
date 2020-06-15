/* External Imports */
import {
  Opcode,
  EVMOpcode,
  EVMOpcodeAndBytes,
  EVMBytecode,
  OpcodeTagReason,
  isValidOpcodeAndBytes,
  Address,
  bytecodeToBuffer,
  formatBytecode,
  getPCOfEVMBytecodeIndex,
} from '@eth-optimism/rollup-core'
import {
  bufToHexString,
  bufferUtils,
  getLogger,
  isValidHexAddress,
  hexStrToBuf,
} from '@eth-optimism/core-utils'

/* Internal Imports */
import { OpcodeReplacer } from '../../types/transpiler'
import {
  InvalidAddressError,
  InvalidBytesConsumedError,
  UnsupportedOpcodeError,
} from '../../index'
import {
  getCALLSubstitute,
  getSTATICCALLSubstitute,
  getDELEGATECALLSubstitute,
  getEXTCODECOPYSubstitute,
} from './dynamic-memory-opcodes'
import {
  getCREATESubstitute,
  getCREATE2Substitute,
} from './contract-creation-opcodes'
import {
  getADDRESSSubstitute,
  getCALLERSubstitute,
  getEXTCODEHASHSubstitute,
  getEXTCODESIZESubstitute,
  getORIGINSubstitute,
  getSLOADSubstitute,
  getSSTORESubstitute,
  getTIMESTAMPSubstitute,
} from './static-memory-opcodes'
import { getPUSHIntegerOp, getPUSHOpcode, isTaggedWithReason } from './helpers'
import { PC_MAX_BYTES } from './constants'

const log = getLogger('transpiler:opcode-replacement')

export class OpcodeReplacerImpl implements OpcodeReplacer {
  public static EX_MGR_PLACEHOLDER: Buffer = Buffer.from(
    `{execution manager address placeholder}`
  )
  private readonly excutionManagerAddressBuffer: Buffer

  /**
   * Creates an OpcodeReplacer, validating the provided address and any given replacements.
   *
   * @param executionManagerAddress The address of the ExecutionManager -- all calls get routed through this contract.
   * @param optionalSubstitutes Optional opcodes to replace with bytecode.
   */
  constructor(
    executionManagerAddress: Address,
    private readonly optionalSubstitutes: Map<
      EVMOpcode,
      EVMBytecode
    > = new Map<EVMOpcode, EVMBytecode>()
  ) {
    // check and store address
    if (!isValidHexAddress(executionManagerAddress)) {
      const msg: string = `Opcode replacer received ${executionManagerAddress} for the execution manager address.  Not a valid hex string address!`
      log.error(msg)
      throw new InvalidAddressError(msg)
    }

    this.excutionManagerAddressBuffer = hexStrToBuf(executionManagerAddress)

    for (const [
      toSubstitute,
      bytecodeToSubstituteWith,
    ] of optionalSubstitutes.entries()) {
      // Make sure we're not attempting to overwrite PUSHN, not yet supported
      if (toSubstitute.programBytesConsumed > 0) {
        const msg: string = `Transpilation currently does not support opcodes which consume bytes, but config specified a substitute for ${JSON.stringify(
          toSubstitute
        )}.`
        log.error(msg)
        throw new UnsupportedOpcodeError(msg)
      }

      // for each operation in the substituted function's bytecode for this toSubstitute...
      for (const substituteBytes of bytecodeToSubstituteWith) {
        // ... replace execution manager placeholder
        if (
          !!substituteBytes.consumedBytes &&
          substituteBytes.consumedBytes.equals(
            OpcodeReplacerImpl.EX_MGR_PLACEHOLDER
          )
        ) {
          substituteBytes.consumedBytes = this.excutionManagerAddressBuffer
        }

        // ...type check consumed bytes are the right length
        if (!isValidOpcodeAndBytes(substituteBytes)) {
          const msg: string = `Replacement config specified a ${
            substituteBytes.opcode.name
          } as an operation in the replacement bytecode for ${
            toSubstitute.name
          }, but the consumed bytes specified was ${bufToHexString(
            substituteBytes.consumedBytes
          )}--invalid length! (length ${substituteBytes.consumedBytes.length})`
          log.error(msg)
          throw new InvalidBytesConsumedError(msg)
        }
      }
    }
  }

  /**
   * Gets whether or not the opcode replacer is configured to change functionality of the given opcode.
   * @param opcodeAndBytes EVM opcode and consumed bytes which might need to be replaced.
   *
   * @returns Whether this opcode needs to get replaced.
   */
  public shouldSubstituteOpcodeForFunction(opcode: EVMOpcode): boolean {
    return (
      !!this.getManadatorySubstitutedFunction({
        opcode,
        consumedBytes: undefined,
      }) || this.optionalSubstitutes.has(opcode)
    )
  }

  /**
   * Gets a chunk of bytecode which will JUMP to the location of the given opcode substitute, and allow JUMPing back on completion
   * @param opcode The opcode whose substitute we should JUMP to
   *
   * @returns The EVMBytecode implementing the above functionality.
   */
  public getJUMPToOpcodeFunction(opcode: EVMOpcode): EVMBytecode {
    return [
      // push the PC to the stack so that we can JUMP back to it
      {
        opcode: Opcode.PC,
        consumedBytes: undefined,
      },
      // JUMP to the right location in the footer
      {
        opcode: getPUSHOpcode(PC_MAX_BYTES),
        consumedBytes: Buffer.alloc(PC_MAX_BYTES),
        tag: {
          padPUSH: false,
          reasonTagged: OpcodeTagReason.IS_PUSH_OPCODE_FUNCTION_LOCATION,
          metadata: opcode,
        },
      },
      {
        opcode: Opcode.JUMP,
        consumedBytes: undefined,
        tag: {
          padPUSH: false,
          reasonTagged: OpcodeTagReason.IS_JUMP_TO_OPCODE_FUNCTION,
          metadata: opcode,
        },
      },
      // allow jumping back once the substituted function was executed
      {
        opcode: Opcode.JUMPDEST,
        consumedBytes: undefined,
        tag: {
          padPUSH: false,
          reasonTagged: OpcodeTagReason.IS_OPCODE_FUNCTION_RETURN_JUMPDEST,
          metadata: undefined,
        },
      },
    ]
  }

  /**
   * Gets a chunk of bytecode which will JUMP back to the original source of execution once the opcode function has been executed.
   * expected stack: [PC of initial opcode which got substituted with getJUMPToOpcodeFunction(...)]
   * @param opcode The opcode whose function is being JUMPed back from.
   *
   * @returns The EVMBytecode implementing the above functionality.
   */
  public getJUMPOnOpcodeFunctionReturn(opcode: EVMOpcode): EVMBytecode {
    // since getJUMPToOpcodeFunction(...)'s first element is the PC, and its last is the JUMPDEST to return to, we need to add its length - 1
    return [
      getPUSHIntegerOp(
        bytecodeToBuffer(this.getJUMPToOpcodeFunction(opcode)).length - 1 // - 1 for the PC opcode
      ),
      {
        opcode: Opcode.ADD,
        consumedBytes: undefined,
      },
      {
        opcode: Opcode.JUMP,
        consumedBytes: undefined,
        tag: {
          padPUSH: false,
          reasonTagged: OpcodeTagReason.IS_OPCODE_FUNCTION_RETURN_JUMP,
          metadata: undefined,
        },
      },
    ]
  }

  /**
   * Gets a piece of bytecode containing substitutes for the given set of opcodes
   * @param opcodeAndBytes The set of opcodes to provide substitutes for in the returned bytcode.
   *
   * @returns Bytecode which can be JUMPed to, executing the opcodes' substitutes, and returning back to the original PC.
   */
  public getOpcodeFunctionTable(opcodes: Set<EVMOpcode>): EVMBytecode {
    const bytecodeToReturn: EVMBytecode = []
    opcodes.forEach((opcode: EVMOpcode) => {
      bytecodeToReturn.push(
        ...[
          // jumpdest to reach
          {
            opcode: Opcode.JUMPDEST,
            consumedBytes: undefined,
            tag: {
              padPUSH: false,
              reasonTagged: OpcodeTagReason.IS_OPCODE_FUNCTION_JUMPDEST,
              metadata: opcode,
            },
          },
          ...this.getSubstituedFunctionFor({ opcode, consumedBytes: undefined }),
          ...this.getJUMPOnOpcodeFunctionReturn(opcode),
        ]
      )
    })
    return bytecodeToReturn
  }

  /**
   * Takes some bytecode which has had opcodes substituted, and the substitute table appended,
   * but with tagged PUSHes of the substitutes jumpdest PC not yet set, and sets them
   * @param taggedBytecode EVM bytecode with some IS_PUSH_OPCODE_FUNCTION_LOCATION tags
   *
   * @returns The final EVMBytecode with the correct PUSH(jumpdest PC) for all substituted jumps.
   */
  public populateOpcodeFunctionJUMPs(taggedBytecode: EVMBytecode): EVMBytecode {
    for (const PUSHOpcodeSubstituteLocation of taggedBytecode.filter((op) =>
      isTaggedWithReason(op, [OpcodeTagReason.IS_PUSH_OPCODE_FUNCTION_LOCATION])
    )) {
      const indexInBytecode = taggedBytecode.findIndex(
        (toCheck: EVMOpcodeAndBytes) => {
          return (
            isTaggedWithReason(toCheck, [
              OpcodeTagReason.IS_OPCODE_FUNCTION_JUMPDEST,
            ]) &&
            toCheck.tag.metadata === PUSHOpcodeSubstituteLocation.tag.metadata
          )
        }
      )
      if (indexInBytecode === -1) {
        throw new Error(
          `unable to find replacment location for opcode ${PUSHOpcodeSubstituteLocation.tag.metadata.name}`
        )
      }
      const PCOfBytecode = getPCOfEVMBytecodeIndex(
        indexInBytecode,
        taggedBytecode
      )
      const destinationBuf = bufferUtils.numberToBuffer(
        PCOfBytecode,
        PC_MAX_BYTES,
        PC_MAX_BYTES
      )
      log.debug(
        `fixed replacement jump with new destination ${bufToHexString(
          destinationBuf
        )}`
      )
      PUSHOpcodeSubstituteLocation.consumedBytes = destinationBuf
    }
    return taggedBytecode
  }

  /**
   * Gets the specified function bytecode meant to be substituted for a given EVM opcode and bytes.
   * The function will be JUMPed to, and back from, in place of executing the un-transpiled opcode.
   * @param opcodeAndBytes EVM opcode and consumed bytes which is supposed to be replaced with JUMPing to the returned function.
   *
   * @returns The EVMBytecode we have decided to substitute the opcodeAndBytes with.
   */
  public getSubstituedFunctionFor(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
    const substitute: EVMBytecode = this.getManadatorySubstitutedFunction(
      opcodeAndBytes
    )
    if (!!substitute) {
      return substitute
    }

    if (!this.optionalSubstitutes.has(opcodeAndBytes.opcode)) {
      return [opcodeAndBytes]
    } else {
      return this.optionalSubstitutes.get(opcodeAndBytes.opcode)
    }
  }

  private getManadatorySubstitutedFunction(
    opcodeAndBytes: EVMOpcodeAndBytes
  ): EVMBytecode {
    switch (opcodeAndBytes.opcode) {
      case Opcode.ADDRESS:
        return getADDRESSSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.CALL:
        return getCALLSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.CALLER:
        return getCALLERSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.CREATE:
        return getCREATESubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.CREATE2:
        return getCREATE2Substitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.DELEGATECALL:
        return getDELEGATECALLSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.EXTCODECOPY:
        return getEXTCODECOPYSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.EXTCODEHASH:
        return getEXTCODEHASHSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.EXTCODESIZE:
        return getEXTCODESIZESubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.ORIGIN:
        return getORIGINSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.SLOAD:
        return getSLOADSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.SSTORE:
        return getSSTORESubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.STATICCALL:
        return getSTATICCALLSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      case Opcode.TIMESTAMP:
        return getTIMESTAMPSubstitute(
          bufToHexString(this.excutionManagerAddressBuffer)
        )
      default:
        return undefined
    }
  }
}
