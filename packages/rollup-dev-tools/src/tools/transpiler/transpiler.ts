/* External Imports */
import {
  Opcode,
  EVMOpcodeAndBytes,
  EVMBytecode,
  bytecodeToBuffer,
  EVMOpcode,
  formatBytecode,
} from '@pigi/rollup-core'
import { getLogger, bufToHexString, hexStrToBuf, add0x } from '@pigi/core-utils'

/* Internal Imports */
import {
  OpcodeWhitelist,
  OpcodeReplacer,
  Transpiler,
  TranspilationResult,
  TranspilationError,
  TranspilationErrors,
} from '../../types/transpiler'
import {
  getExpectedFooterSwitchStatementJumpdestIndex,
  getJumpIndexSwitchStatementBytecode,
  getJumpiReplacementBytecode,
  getJumpiReplacementBytecodeLength,
  getJumpReplacementBytecode,
  getJumpReplacementBytecodeLength,
} from './jump-replacement'

const log = getLogger('transpiler-impl')

export class TranspilerImpl implements Transpiler {
  constructor(
    private readonly opcodeWhitelist: OpcodeWhitelist,
    private readonly opcodeReplacer: OpcodeReplacer
  ) {
    if (!opcodeWhitelist) {
      throw Error('Opcode Whitelist is required for TranspilerImpl')
    }
    if (!opcodeReplacer) {
      throw Error('Opcode Replacer is required for TranspilerImpl')
    }
  }

  public transpile(inputBytecode: Buffer): TranspilationResult {
    let transpiledBytecode: EVMBytecode = []
    const errors: TranspilationError[] = []
    const jumpdestIndexesBefore: number[] = []
    let lastOpcode: EVMOpcode
    let insideUnreachableCode: boolean = false
    let seenJump: boolean = false
    for (let pc = 0; pc < inputBytecode.length; pc++) {
      let opcode = Opcode.parseByNumber(inputBytecode[pc])
      // This JUMPDEST is reachable
      if (insideUnreachableCode && seenJump && opcode === Opcode.JUMPDEST) {
        insideUnreachableCode = false
      }
      if (!insideUnreachableCode) {
        if (
          !TranspilerImpl.validOpcode(
            opcode,
            pc,
            inputBytecode[pc],
            lastOpcode,
            errors
          )
        ) {
          lastOpcode = undefined
          continue
        }
        lastOpcode = opcode
        seenJump = seenJump || Opcode.JUMP_OP_CODES.includes(opcode)
        insideUnreachableCode = Opcode.HALTING_OP_CODES.includes(opcode)

        if (opcode === Opcode.JUMPDEST) {
          jumpdestIndexesBefore.push(pc)
        }
        if (!this.opcodeWhitelisted(opcode, pc, errors)) {
          pc += opcode.programBytesConsumed
          continue
        }
        if (
          !TranspilerImpl.enoughBytesLeft(
            opcode,
            inputBytecode.length,
            pc,
            errors
          )
        ) {
          break
        }
      }
      if (insideUnreachableCode && !opcode) {
        const unreachableCode: Buffer = inputBytecode.slice(pc, pc + 1)
        opcode = {
          name: `UNREACHABLE (${bufToHexString(unreachableCode)})`,
          code: unreachableCode,
          programBytesConsumed: 0,
        }
      }

      const opcodeAndBytes: EVMOpcodeAndBytes = {
        opcode,
        consumedBytes: !opcode.programBytesConsumed
          ? undefined
          : inputBytecode.slice(pc + 1, pc + 1 + opcode.programBytesConsumed),
      }
      // copy over opcode as is if unreachable
      const transpiledOpcodeAndBytes = insideUnreachableCode
        ? [opcodeAndBytes]
        : this.opcodeReplacer.replaceIfNecessary(opcodeAndBytes)

      transpiledBytecode.push(...transpiledOpcodeAndBytes)
      pc += opcode.programBytesConsumed
    }

    log.debug(
      `Bytecode after replacement before JUMP logic: \n${formatBytecode(
        transpiledBytecode
      )}`
    )

    transpiledBytecode = TranspilerImpl.accountForJumps(
      transpiledBytecode,
      jumpdestIndexesBefore,
      errors
    )

    if (!!errors.length) {
      return {
        succeeded: false,
        errors,
      }
    }
    return {
      succeeded: true,
      bytecode: bytecodeToBuffer(transpiledBytecode),
    }
  }

  /**
   * Returns whether or not the provided EVMOpcode is valid (not undefined).
   * If it is not, it creates a new TranpilationError and appends it to the provided list.
   *
   * @param opcode The opcode in question.
   * @param pc The current program counter value.
   * @param code The code (decimal) of the opcode in question .
   * @param lastOpcode The last Opcode seen before this one.
   * @param errors The cumulative errors list.
   * @returns True if valid, False otherwise.
   */
  private static validOpcode(
    opcode: EVMOpcode,
    pc: number,
    code: number,
    lastOpcode: EVMOpcode,
    errors: TranspilationError[]
  ): boolean {
    if (!opcode) {
      let messageExtension: string = ''
      if (!!lastOpcode && !!lastOpcode.programBytesConsumed) {
        messageExtension = ` Was ${lastOpcode.name} at index ${pc -
          lastOpcode.programBytesConsumed} provided exactly ${
          lastOpcode.programBytesConsumed
        } bytes as expected?`
      }
      const message: string = `Cannot find opcode for: ${add0x(
        code.toString(16)
      )}.${messageExtension}`
      log.debug(message)
      errors.push(
        TranspilerImpl.createError(
          pc,
          TranspilationErrors.UNSUPPORTED_OPCODE,
          message
        )
      )
      return false
    }
    return true
  }

  /**
   * Returns whether or not the provided EVMOpcode is whitelisted.
   * If it is not, it creates a new TranpilationError and appends it to the provided list.
   *
   * @param opcode The opcode in question.
   * @param pc The current program counter value.
   * @param errors The cumulative errors list.
   * @returns True if whitelisted, False otherwise.
   */
  private opcodeWhitelisted(
    opcode: EVMOpcode,
    pc: number,
    errors: TranspilationError[]
  ): boolean {
    if (!this.opcodeWhitelist.isOpcodeWhitelisted(opcode)) {
      const message: string = `Opcode [${opcode.name}] is not on the whitelist.`
      log.debug(message)
      errors.push(
        TranspilerImpl.createError(
          pc,
          TranspilationErrors.OPCODE_NOT_WHITELISTED,
          message
        )
      )
      return false
    }
    return true
  }

  /**
   * Returns whether or not there are enough bytes left in the bytecode for the provided Opcode.
   * If it is not, it creates a new TranpilationError and appends it to the provided list.
   *
   * @param opcode The opcode in question.
   * @param bytecodeLength The length of the bytecode being transpiled.
   * @param pc The current program counter value.
   * @param errors The cumulative errors list.
   * @returns True if enough bytes are left for the Opcode to consume, False otherwise.
   */
  private static enoughBytesLeft(
    opcode: EVMOpcode,
    bytecodeLength: number,
    pc: number,
    errors: TranspilationError[]
  ): boolean {
    if (pc + opcode.programBytesConsumed >= bytecodeLength) {
      const bytesLeft: number = bytecodeLength - pc - 1
      const message: string = `Opcode: ${opcode.name} consumes ${
        opcode.programBytesConsumed
      }, but ${!!bytesLeft ? 'only ' : ''}${bytesLeft} ${
        bytesLeft !== 1 ? 'bytes are' : 'byte is'
      } left in input bytecode.`
      log.debug(message)
      errors.push(
        TranspilerImpl.createError(
          pc,
          TranspilationErrors.INVALID_BYTES_CONSUMED,
          message
        )
      )
      return false
    }
    return true
  }

  /**
   * Takes the provided transpiled bytecode and accounts for JUMPs that may not jump
   * to the intended spots now that transpilation has modified the code.
   *
   * @param transpiledBytecode The transpiled bytecode to operate on.
   * @param jumpdestIndexesBefore The ordered indexes of JUMPDESTs before.
   * @param errors The list of errors to append to if there is an error.
   * @returns The new bytecode with all JUMPs accounted for.
   */
  private static accountForJumps(
    transpiledBytecode: EVMBytecode,
    jumpdestIndexesBefore: number[],
    errors: TranspilationError[]
  ): EVMBytecode {
    if (jumpdestIndexesBefore.length === 0) {
      return transpiledBytecode
    }

    const footerSwitchJumpdestIndex: number = getExpectedFooterSwitchStatementJumpdestIndex(
      transpiledBytecode
    )
    const jumpdestIndexesAfter: number[] = []
    const replacedBytecode: EVMBytecode = []
    let pc: number = 0
    // Replace all JUMP, JUMPI, and JUMPDEST, and build the post-transpilation JUMPDEST index array.
    for (const opcodeAndBytes of transpiledBytecode) {
      if (opcodeAndBytes.opcode === Opcode.JUMP) {
        replacedBytecode.push(
          ...getJumpReplacementBytecode(footerSwitchJumpdestIndex)
        )
        pc += getJumpReplacementBytecodeLength()
      } else if (opcodeAndBytes.opcode === Opcode.JUMPI) {
        replacedBytecode.push(
          ...getJumpiReplacementBytecode(footerSwitchJumpdestIndex)
        )
        pc += getJumpiReplacementBytecodeLength()
      } else if (opcodeAndBytes.opcode === Opcode.JUMPDEST) {
        replacedBytecode.push(opcodeAndBytes)
        jumpdestIndexesAfter.push(pc)
        pc += 1
      } else {
        replacedBytecode.push(opcodeAndBytes)
        pc += 1 + opcodeAndBytes.opcode.programBytesConsumed
      }
    }

    if (jumpdestIndexesBefore.length !== jumpdestIndexesAfter.length) {
      const message: string = `There were ${jumpdestIndexesBefore.length} JUMPDESTs before transpilation, but there are ${jumpdestIndexesAfter.length} JUMPDESTs after.`
      log.debug(message)
      errors.push(
        TranspilerImpl.createError(
          -1,
          TranspilationErrors.INVALID_SUBSTITUTION,
          message
        )
      )
      return transpiledBytecode
    }

    // Add the logic to handle the pre-transpilation to post-transpilation jump dest mapping.
    replacedBytecode.push(
      ...getJumpIndexSwitchStatementBytecode(
        jumpdestIndexesBefore,
        jumpdestIndexesAfter,
        bytecodeToBuffer(replacedBytecode).length
      )
    )

    return replacedBytecode
  }

  /**
   * Util function to create TranspilationErrors.
   *
   * @param index The index of the byte in the input bytecode where the error originates.
   * @param error The TranspilationErrors error type.
   * @param message The error message.
   * @returns The constructed TranspilationError
   */
  private static createError(
    index: number,
    error: number,
    message: string
  ): TranspilationError {
    return {
      index,
      error,
      message,
    }
  }
}
