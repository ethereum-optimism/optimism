import { should } from '../setup'

/* External Imports */
import {
  bufferUtils,
  bufToHexString,
  getLogger,
} from '@eth-optimism/core-utils'

import {
  Opcode,
  EVMOpcode,
  EVMBytecode,
  bytecodeToBuffer,
} from '@eth-optimism/rollup-core'

/* Internal imports */
import {
  SuccessfulTranspilation,
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
  invalidBytesConsumedBytecodeNoReturn,
  invalidOpcode,
  multipleErrors,
  multipleNonWhitelisted,
  singleNonWhitelisted,
  stateManagerAddress,
  validBytecode,
  whitelistedOpcodes,
} from '../helpers'

const log = getLogger(`transpile`)
const haltingOpcodes: EVMOpcode[] = Opcode.HALTING_OP_CODES
const haltingOpcodesNoJump: EVMOpcode[] = haltingOpcodes.filter(
  (x) => x.name !== 'JUMP'
)
const jumps: EVMOpcode[] = Opcode.JUMP_OP_CODES

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
      const result: TranspilationResult = transpiler.transpileRawBytecode(
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

      const result: TranspilationResult = transpiler.transpileRawBytecode(
        inputBytecode
      )

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

      const result: TranspilationResult = transpiler.transpileRawBytecode(
        inputBytecode
      )

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
      const result: TranspilationResult = transpiler.transpileRawBytecode(
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
      const result: TranspilationResult = transpiler.transpileRawBytecode(
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
      const bytecode: Buffer = bytecodeToBuffer(
        invalidBytesConsumedBytecodeNoReturn
      )

      const result: TranspilationResult = transpiler.transpileRawBytecode(
        bytecode
      )
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

      const result: TranspilationResult = transpiler.transpileRawBytecode(
        bytecode
      )
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

      const result: TranspilationResult = transpiler.transpileRawBytecode(
        bytecode
      )
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

      const result: TranspilationResult = transpiler.transpileRawBytecode(
        bytecode
      )
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

  describe('handles unreachable code', async () => {
    it(`skips unreachable bytecode after a halting opcode`, async () => {
      for (const haltingOp of haltingOpcodes) {
        const bytecode: Buffer = Buffer.concat([haltingOp.code, invalidOpcode])
        const result: TranspilationResult = transpiler.transpileRawBytecode(
          bytecode
        )
        result.succeeded.should.equal(
          true,
          `Bytecode containing invalid opcodes in unreachable code after a ${haltingOp.name} should not have failed!`
        )
        const success = result as SuccessfulTranspilation
        success.bytecode.should.eql(
          bytecode,
          `Bytecode containing invalid opcodes in unreachable code after a ${haltingOp.name} should not have changed any bytecode!`
        )
      }
    })
    it('skips bytecode after an unreachable JUMPDEST', async () => {
      for (const haltingOp of haltingOpcodesNoJump) {
        const bytecode: Buffer = Buffer.concat([
          haltingOp.code,
          invalidOpcode,
          Opcode.JUMPDEST.code,
          invalidOpcode,
        ])
        const result: TranspilationResult = transpiler.transpileRawBytecode(
          bytecode
        )
        result.succeeded.should.equal(
          true,
          `Bytecode containing invalid opcodes in unreachable code after unreachable JUMPDEST (after a ${haltingOp.name})should not have failed!`
        )
        const success = result as SuccessfulTranspilation
        success.bytecode.should.eql(
          bytecode,
          `Bytecode containing invalid opcodes in unreachable code after unreachable JUMPDEST (after a ${haltingOp.name}) should not have changed any bytecode!`
        )
      }
    })
    it('parses opcodes after a reachable JUMPDEST', async () => {
      for (const haltingOp of haltingOpcodesNoJump) {
        for (const jump of jumps) {
          const bytecode: Buffer = Buffer.concat([
            jump.code,
            // JUMPDEST here so that the haltingOp is reachable
            Opcode.JUMPDEST.code,
            haltingOp.code,
            Opcode.JUMPDEST.code,
            invalidOpcode,
          ])
          const result: TranspilationResult = transpiler.transpileRawBytecode(
            bytecode
          )
          result.succeeded.should.equal(
            false,
            `Bytecode containing invalid opcodes after reachable JUMPDEST preceded by a ${haltingOp.name} should have failed!`
          )
          const error: ErroredTranspilation = result as ErroredTranspilation
          error.errors.length.should.equal(1)
          error.errors[0].index.should.equal(bytecode.length - 1)
          error.errors[0].error.should.equal(
            TranspilationErrors.UNSUPPORTED_OPCODE
          )
        }
      }
    })
    it('parses opcodes after first JUMP and JUMPDEST', async () => {
      const bytecode: Buffer = Buffer.concat([
        Opcode.JUMP.code,
        Opcode.JUMPDEST.code,
        invalidOpcode,
      ])
      const result: TranspilationResult = transpiler.transpileRawBytecode(
        bytecode
      )
      result.succeeded.should.equal(
        false,
        `Bytecode containing invalid opcodes after reachable JUMPDEST should have failed!`
      )
      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(1)
      error.errors[0].index.should.equal(bytecode.length - 1)
      error.errors[0].error.should.equal(TranspilationErrors.UNSUPPORTED_OPCODE)
    })

    it('parses opcodes after JUMPI', async () => {
      const bytecode: Buffer = Buffer.concat([Opcode.JUMPI.code, invalidOpcode])
      const result: TranspilationResult = transpiler.transpileRawBytecode(
        bytecode
      )
      result.succeeded.should.equal(
        false,
        `Bytecode containing invalid opcodes after reachable JUMPI should have failed!`
      )
      const error: ErroredTranspilation = result as ErroredTranspilation
      error.errors.length.should.equal(1)
      error.errors[0].index.should.equal(bytecode.length - 1)
      error.errors[0].error.should.equal(TranspilationErrors.UNSUPPORTED_OPCODE)
    })

    it('should correctly handle alternating reachable/uncreachable code ending in reachable, valid code', async () => {
      for (const haltingOp of haltingOpcodesNoJump) {
        for (const jump of jumps) {
          let bytecode: Buffer = Buffer.concat([
            jump.code,
            // JUMPDEST here so that the haltingOp is reachable
            Opcode.JUMPDEST.code,
          ])
          for (let i = 0; i < 3; i++) {
            bytecode = Buffer.concat([
              bytecode,
              haltingOp.code,
              // Unreachable, invalid code
              invalidOpcode,
              Opcode.JUMPDEST.code,
              // Reachable, valid code
              Opcode.ADD.code,
            ])
          }
          const result: TranspilationResult = transpiler.transpileRawBytecode(
            bytecode
          )
          result.succeeded.should.equal(
            true,
            `Long bytecode containing alternating valid reachable and invalid unreachable code failed!`
          )
        }
      }
    })

    it('should correctly handle alternating reachable/uncreachable code ending in reachable, invalid code', async () => {
      for (const haltingOp of haltingOpcodesNoJump) {
        for (const jump of jumps) {
          let bytecode: Buffer = Buffer.concat([
            jump.code,
            // JUMPDEST here so that the haltingOp is reachable
            Opcode.JUMPDEST.code,
          ])
          for (let i = 0; i < 3; i++) {
            bytecode = Buffer.concat([
              bytecode,
              haltingOp.code,
              // Unreachable, invalid code
              invalidOpcode,
              Opcode.JUMPDEST.code,
              // Reachable, valid code
              Opcode.ADD.code,
            ])
          }
          bytecode = Buffer.concat([
            bytecode,
            // Reachable, invalid code
            invalidOpcode,
          ])
          const result: TranspilationResult = transpiler.transpileRawBytecode(
            bytecode
          )
          result.succeeded.should.equal(
            false,
            `Long bytecode ending in reachable, invalid code should have failed!`
          )
          const error: ErroredTranspilation = result as ErroredTranspilation
          error.errors.length.should.equal(1)
          error.errors[0].index.should.equal(bytecode.length - 1)
          error.errors[0].error.should.equal(
            TranspilationErrors.UNSUPPORTED_OPCODE
          )
        }
      }
    })
  })
})
