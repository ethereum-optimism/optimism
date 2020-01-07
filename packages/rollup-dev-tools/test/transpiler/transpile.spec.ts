import { should } from '../setup'

/* External Imports */
import { bufferUtils, bufToHexString } from '@pigi/core-utils'
import {
  Opcode,
  EVMOpcode,
  EVMBytecode,
  bytecodeToBuffer,
} from '@pigi/rollup-core'

/* Internal imports */
import {
  ErroredTranspilation,
  OpcodeReplacer,
  OpcodeWhitelist,
  TranspilationErrors,
  TranspilationResult,
  Transpiler,
} from '../../src/types/transpiler'
import {
  TranspilerImpl,
  OpcodeReplacerImpl,
  OpcodeWhitelistImpl,
} from '../../src/tools/transpiler'
import {
  invalidBytesConsumedBytecode,
  invalidOpcode,
  multipleErrors,
  multipleNonWhitelisted,
  singleNonWhitelisted,
  stateManagerAddress,
  validBytecode,
  whitelistedOpcodes,
} from '../helpers'

describe('Transpile', () => {
  let opcodeWhitelist: OpcodeWhitelist
  let transpiler: Transpiler
  let replacer: OpcodeReplacer

  beforeEach(() => {
    opcodeWhitelist = new OpcodeWhitelistImpl(whitelistedOpcodes)
    replacer = new OpcodeReplacerImpl(
      stateManagerAddress,
      new Map<EVMOpcode, EVMBytecode>()
    )
    transpiler = new TranspilerImpl(opcodeWhitelist, replacer)
  })

  describe('Valid input', () => {
    it('correctly accepts valid bytecode input', () => {
      const result: TranspilationResult = transpiler.transpile(
        bytecodeToBuffer(validBytecode)
      )
      result.succeeded.should.equal(true)
    })
  })

  describe('Unsupported Opcodes', () => {
    it('flags unsupported opcode', () => {
      const inputBytecode: Buffer = Buffer.concat([
        bytecodeToBuffer(validBytecode),
        invalidOpcode,
      ])

      const result: TranspilationResult = transpiler.transpile(inputBytecode)

      result.succeeded.should.equal(false)

      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(1)
      error.errors[0].index.should.equal(inputBytecode.length - 1)
      error.errors[0].error.should.equal(TranspilationErrors.UNSUPPORTED_OPCODE)
    })

    it('flags multiple unsupported opcodes', () => {
      const inputBytecode: Buffer = Buffer.concat([
        invalidOpcode,
        bytecodeToBuffer(validBytecode),
        invalidOpcode,
      ])

      const result: TranspilationResult = transpiler.transpile(inputBytecode)

      result.succeeded.should.equal(false)

      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(2)
      error.errors[0].index.should.equal(0)
      error.errors[0].error.should.equal(TranspilationErrors.UNSUPPORTED_OPCODE)
      error.errors[1].index.should.equal(inputBytecode.length - 1)
      error.errors[1].error.should.equal(TranspilationErrors.UNSUPPORTED_OPCODE)
    })
  })

  describe('Whitelist Enforcement', () => {
    it('flags non-whitelisted opcode', () => {
      const result: TranspilationResult = transpiler.transpile(
        bytecodeToBuffer(singleNonWhitelisted)
      )
      result.succeeded.should.equal(false)

      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(1)
      error.errors[0].index.should.equal(8)
      error.errors[0].error.should.equal(
        TranspilationErrors.OPCODE_NOT_WHITELISTED
      )
    })

    it('flags multiple non-whitelisted opcode', () => {
      const result: TranspilationResult = transpiler.transpile(
        bytecodeToBuffer(multipleNonWhitelisted)
      )
      result.succeeded.should.equal(false)

      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(2)
      error.errors[0].index.should.equal(8)
      error.errors[0].error.should.equal(
        TranspilationErrors.OPCODE_NOT_WHITELISTED
      )

      error.errors[1].index.should.equal(12)
      error.errors[1].error.should.equal(
        TranspilationErrors.OPCODE_NOT_WHITELISTED
      )
    })
  })

  describe('Enforces Invalid Bytes Consumed', () => {
    it('flags invalid bytes consumed', () => {
      const bytecode: Buffer = bytecodeToBuffer(invalidBytesConsumedBytecode)

      const result: TranspilationResult = transpiler.transpile(bytecode)
      result.succeeded.should.equal(false)

      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(1)
      error.errors[0].index.should.equal(bytecode.length - 1)
      error.errors[0].error.should.equal(
        TranspilationErrors.INVALID_BYTES_CONSUMED
      )
    })

    it('flags invalid (less) bytes consumed as unrecognized opcode', () => {
      const bytecode: Buffer = bytecodeToBuffer([
        {
          opcode: Opcode.PUSH1,
          consumedBytes: undefined,
        },
        {
          opcode: Opcode.PUSH1,
          consumedBytes: invalidOpcode,
        },
      ])

      const result: TranspilationResult = transpiler.transpile(bytecode)
      result.succeeded.should.equal(false)

      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(1)
      error.errors[0].index.should.equal(2)
      error.errors[0].error.should.equal(TranspilationErrors.UNSUPPORTED_OPCODE)
      error.errors[0].message.endsWith('?').should.equal(true)
    })

    it('flags invalid (more) bytes consumed as unrecognized opcode', () => {
      const bytecode: Buffer = bytecodeToBuffer([
        {
          opcode: Opcode.PUSH1,
          consumedBytes: Buffer.concat([
            Buffer.from('00', 'hex'),
            invalidOpcode,
          ]),
        },
        {
          opcode: Opcode.PUSH1,
          consumedBytes: invalidOpcode,
        },
      ])

      const result: TranspilationResult = transpiler.transpile(bytecode)
      result.succeeded.should.equal(false)

      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(1)
      error.errors[0].index.should.equal(2)
      error.errors[0].error.should.equal(TranspilationErrors.UNSUPPORTED_OPCODE)
      error.errors[0].message.endsWith('?').should.equal(true)
    })
  })

  describe('Multiple Errors', () => {
    it('flags all error types at once', () => {
      const bytecode: Buffer = Buffer.concat([
        invalidOpcode,
        bytecodeToBuffer(multipleErrors),
      ])

      const result: TranspilationResult = transpiler.transpile(bytecode)
      result.succeeded.should.equal(false)

      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(3)
      error.errors[0].index.should.equal(0)
      error.errors[0].error.should.equal(TranspilationErrors.UNSUPPORTED_OPCODE)
      error.errors[1].index.should.equal(bytecode.length - 2)
      error.errors[1].error.should.equal(
        TranspilationErrors.OPCODE_NOT_WHITELISTED
      )
      error.errors[2].index.should.equal(bytecode.length - 1)
      error.errors[2].error.should.equal(
        TranspilationErrors.INVALID_BYTES_CONSUMED
      )
    })
  })
})
