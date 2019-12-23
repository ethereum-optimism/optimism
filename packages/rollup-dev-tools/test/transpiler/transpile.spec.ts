import '../setup'

/* External Imports */
import {
  Opcode,
  EVMOpcode,
  Address,
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
} from '../../src/transpiler'

const wOpcodes: EVMOpcode[] = [
  Opcode.PUSH1,
  Opcode.PUSH4,
  Opcode.PUSH29,
  Opcode.MSTORE,
  Opcode.CALLDATALOAD,
  Opcode.SWAP1,
  Opcode.SWAP2,
  Opcode.SWAP3,
  Opcode.DIV,
  Opcode.DUP1,
  Opcode.DUP2,
  Opcode.DUP3,
  Opcode.DUP4,
  Opcode.EQ,
  Opcode.JUMPI,
  Opcode.JUMP,
  Opcode.JUMPDEST,
  Opcode.STOP,
  Opcode.ADD,
  Opcode.MUL,
  Opcode.POP,
  Opcode.MLOAD,
  Opcode.SUB,
  Opcode.RETURN,
]

const stateManager: Address = '0x0000000000000000000000000000000000000000'

const validBytecode: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },
  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
  { opcode: Opcode.RETURN, consumedBytes: undefined },
]

const singleNonWhitelisted: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },

  { opcode: Opcode.SSTORE, consumedBytes: undefined },

  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
  { opcode: Opcode.RETURN, consumedBytes: undefined },
]

const multipleNonWhitelisted: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },

  { opcode: Opcode.SSTORE, consumedBytes: undefined },

  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },

  { opcode: Opcode.SLOAD, consumedBytes: undefined },

  { opcode: Opcode.RETURN, consumedBytes: undefined },
]

const invalidBytesConsumedBytecode: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },
  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
  { opcode: Opcode.RETURN, consumedBytes: undefined },
  { opcode: Opcode.PUSH1, consumedBytes: undefined },
]

const multipleErrors: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },
  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
  { opcode: Opcode.RETURN, consumedBytes: undefined },
  { opcode: Opcode.SLOAD, consumedBytes: undefined },
  { opcode: Opcode.PUSH1, consumedBytes: undefined },
]

const invalidOpcode: Buffer = Buffer.from('5d', 'hex')

describe('Transpile', () => {
  let opcodeWhitelist: OpcodeWhitelist
  let transpiler: Transpiler
  let replacer: OpcodeReplacer

  beforeEach(() => {
    opcodeWhitelist = new OpcodeWhitelistImpl(wOpcodes)
    replacer = new OpcodeReplacerImpl(
      stateManager,
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
