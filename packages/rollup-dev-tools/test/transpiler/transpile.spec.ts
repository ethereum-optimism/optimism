import { should } from '../setup'

/* External Imports */
import { bufferUtils, bufToHexString } from '@pigi/core-utils'
import {
  Opcode,
  EVMOpcode,
  Address,
  EVMBytecode,
  bytecodeToBuffer,
  bufferToBytecode,
  EVMOpcodeAndBytes,
} from '@pigi/rollup-core'

/* Internal imports */
import {
  ErroredTranspilation,
  OpcodeReplacer,
  OpcodeWhitelist,
  SuccessfulTranspilation,
  TranspilationErrors,
  TranspilationResult,
  Transpiler,
} from '../../src/types/transpiler'
import {
  TranspilerImpl,
  OpcodeReplacerImpl,
  OpcodeWhitelistImpl,
} from '../../src/transpiler'

const whitelistedOpcodes: EVMOpcode[] = [
  Opcode.PUSH1,
  Opcode.PUSH4,
  Opcode.PUSH29,
  Opcode.PUSH32,
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

const validateJumpBytecode = (
  transpiler: Transpiler,
  bytecode: EVMBytecode
) => {
  const result: TranspilationResult = transpiler.transpile(
    bytecodeToBuffer(bytecode)
  )

  result.succeeded.should.equal(true)
  const successResult: SuccessfulTranspilation = result as SuccessfulTranspilation
  const outputBytecode: EVMBytecode = bufferToBytecode(successResult.bytecode)

  let pc = 0
  let lastOpcode: EVMOpcodeAndBytes
  const jumpdestIndexes: number[] = []
  const opcodesBeforeJump: Map<number, EVMOpcodeAndBytes> = new Map<
    number,
    EVMOpcodeAndBytes
  >()
  // Build map of index => opcode immediately before JUMP and get index of footer switch
  for (const opcodeAndBytes of outputBytecode) {
    if (opcodeAndBytes.opcode === Opcode.JUMPDEST) {
      jumpdestIndexes.push(pc)
    }
    if (
      opcodeAndBytes.opcode === Opcode.JUMP ||
      opcodeAndBytes.opcode === Opcode.JUMPI
    ) {
      opcodesBeforeJump.set(
        pc - 1 - lastOpcode.opcode.programBytesConsumed,
        lastOpcode
      )
    }
    lastOpcode = opcodeAndBytes
    pc += 1 + opcodeAndBytes.opcode.programBytesConsumed
  }

  jumpdestIndexes.length.should.be.greaterThan(
    0,
    'There should be JUMPDESTs, but there are not!'
  )

  const switchJumpdestIndex: number = jumpdestIndexes.pop()
  const switchJumpdest: Buffer = successResult.bytecode.slice(
    switchJumpdestIndex,
    switchJumpdestIndex + 1
  )
  switchJumpdest.should.eql(
    Opcode.JUMPDEST.code,
    `Switch JUMPDEST index is ${switchJumpdestIndex}, but byte at that index is ${bufToHexString(
      switchJumpdest
    )}, not ${bufToHexString(Opcode.JUMPDEST.code)}`
  )

  opcodesBeforeJump.size.should.be.greaterThan(
    0,
    'opcodesBeforeJump should have entries but does not!'
  )

  for (const [index, opcodeBeforeJump] of opcodesBeforeJump.entries()) {
    opcodeBeforeJump.opcode.should.equal(
      Opcode.PUSH32,
      'Opcode before JUMP should be a PUSH32, pushing the location of the footer JUMP switch!'
    )
    if (index < switchJumpdestIndex) {
      // All regular program JUMPs should go to the footer JUMPDEST
      opcodeBeforeJump.consumedBytes.should.eql(
        bufferUtils.numberToBuffer(switchJumpdestIndex),
        'JUMP should be equal to index of footer switch JUMPDEST!'
      )
    } else {
      // Make sure that all footer JUMPS go to JUMPDESTs
      const dest: number = opcodeBeforeJump.consumedBytes.readInt32BE(28)
      successResult.bytecode
        .slice(dest, dest + 1)
        .should.eql(
          Opcode.JUMPDEST.code,
          'JUMP should be equal to index of footer switch JUMPDEST!'
        )
    }
  }

  // Need to make sure that regular program JUMPDESTs are followed by POP
  // due to the way our switch statement leaves an extra item on the stack.
  for (const index of jumpdestIndexes) {
    successResult.bytecode
      .slice(index + 1, index + 2)
      .should.eql(Opcode.POP.code)
  }
}

describe('Transpile', () => {
  let opcodeWhitelist: OpcodeWhitelist
  let transpiler: Transpiler
  let replacer: OpcodeReplacer

  beforeEach(() => {
    opcodeWhitelist = new OpcodeWhitelistImpl(whitelistedOpcodes)
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

  describe('JUMPs and the like', () => {
    it('handles simple JUMPs properly', () => {
      const simpleJumpBytecode: EVMBytecode = [
        { opcode: Opcode.JUMP, consumedBytes: undefined },
        { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
      ]

      validateJumpBytecode(transpiler, simpleJumpBytecode)
    })

    it('handles simple JUMPIs properly', () => {
      const simpleJumpBytecode: EVMBytecode = [
        { opcode: Opcode.PUSH32, consumedBytes: bufferUtils.numberToBuffer(1) },
        {
          opcode: Opcode.PUSH32,
          consumedBytes: bufferUtils.numberToBuffer(67),
        },
        { opcode: Opcode.JUMPI, consumedBytes: undefined },
        { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
      ]

      validateJumpBytecode(transpiler, simpleJumpBytecode)
    })

    it('handles complex JUMP(I)s properly', () => {
      const simpleJumpBytecode: EVMBytecode = [
        { opcode: Opcode.PUSH32, consumedBytes: bufferUtils.numberToBuffer(1) },
        {
          opcode: Opcode.PUSH32,
          consumedBytes: bufferUtils.numberToBuffer(104),
        },
        { opcode: Opcode.JUMPI, consumedBytes: undefined },
        { opcode: Opcode.PUSH1, consumedBytes: bufferUtils.numberToBuffer(1) },
        { opcode: Opcode.DUP1, consumedBytes: undefined },
        { opcode: Opcode.SWAP1, consumedBytes: undefined },
        { opcode: Opcode.DIV, consumedBytes: undefined },
        { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
        { opcode: Opcode.MUL, consumedBytes: undefined },
        { opcode: Opcode.RETURN, consumedBytes: undefined },
        { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
        {
          opcode: Opcode.PUSH32,
          consumedBytes: bufferUtils.numberToBuffer(107),
        },
        { opcode: Opcode.JUMP, consumedBytes: bufferUtils.numberToBuffer(1) },
      ]

      validateJumpBytecode(transpiler, simpleJumpBytecode)
    })

    it('handles code without JUMPs properly', () => {
      const simpleJumpBytecode: EVMBytecode = [
        {
          opcode: Opcode.PUSH32,
          consumedBytes: bufferUtils.numberToBuffer(67),
        },
        { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
      ]

      validateJumpBytecode(transpiler, simpleJumpBytecode)
    })

    it('handles code without JUMPs or JUMPDESTs properly', () => {
      const simpleJumpBytecode: EVMBytecode = [
        {
          opcode: Opcode.PUSH32,
          consumedBytes: bufferUtils.numberToBuffer(67),
        },
      ]

      const bufferPreTranspilation: Buffer = bytecodeToBuffer(
        simpleJumpBytecode
      )
      const result: TranspilationResult = transpiler.transpile(
        bufferPreTranspilation
      )
      result.succeeded.should.equal(
        true,
        'Transpilation should have succeeded but did not!'
      )

      const successResult: SuccessfulTranspilation = result as SuccessfulTranspilation

      successResult.bytecode.should.eql(
        bufferPreTranspilation,
        'Transpilation should not have changed anything but did!'
      )
    })
  })
})
