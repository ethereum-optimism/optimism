/* External Imports */
import {
  Opcode,
  Address,
  EVMOpcodeAndBytes,
  EVMBytecode,
  bytecodeToBuffer,
  EVMOpcode,
} from '@pigi/rollup-core'
import { getLogger } from '@pigi/core-utils'

/* Internal Imports */
import {
  OpcodeWhitelist,
  OpcodeReplacer,
  Transpiler,
  TranspilationResult,
  TranspilationError,
  TranspilationErrors,
} from '../types/transpiler'

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
    const errors: TranspilationError[] = []
    const transpiledBytecode: EVMBytecode = []
    let lastOpcode: EVMOpcode
    for (let pc = 0; pc < inputBytecode.length; pc++) {
      const opcode = Opcode.parseByNumber(inputBytecode[pc])

      if (!this.validOpcode(opcode, pc, lastOpcode, errors)) {
        lastOpcode = undefined
        continue
      }
      lastOpcode = opcode

      if (!this.opcodeWhitelisted(opcode, pc, errors)) {
        pc += opcode.programBytesConsumed
        continue
      }
      if (!this.enoughBytesLeft(opcode, inputBytecode.length, pc, errors)) {
        break
      }

      // Replacement
      const opcodeAndBytes: EVMOpcodeAndBytes = {
        opcode,
        consumedBytes: !opcode.programBytesConsumed
          ? undefined
          : inputBytecode.slice(pc, pc + opcode.programBytesConsumed),
      }

      transpiledBytecode.push(
        ...this.opcodeReplacer.replaceIfNecessary(opcodeAndBytes)
      )

      pc += opcode.programBytesConsumed
    }

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
   * @param lastOpcode The last Opcode seen before this one.
   * @param errors The cumulative errors list.
   * @returns True if valid, False otherwise.
   */
  private validOpcode(
    opcode: EVMOpcode,
    pc: number,
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
      const message: string = `Cannot find opcode for number (decimal): ${opcode}.${messageExtension}`
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
  private enoughBytesLeft(
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
