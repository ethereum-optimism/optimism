import { should } from '../setup'

/* External Imports */
import { bufferUtils, bufToHexString } from '@pigi/core-utils'
import {
  Opcode,
  EVMOpcode,
  EVMBytecode,
  bytecodeToBuffer,
  bufferToBytecode,
  EVMOpcodeAndBytes,
  formatBytecode,
} from '@pigi/rollup-core'

/* Internal imports */
import {
  OpcodeReplacer,
  OpcodeWhitelist,
  SuccessfulTranspilation,
  TranspilationResult,
  Transpiler,
} from '../../src/types/transpiler'
import {
  TranspilerImpl,
  OpcodeReplacerImpl,
  OpcodeWhitelistImpl,
} from '../../src/tools/transpiler'
import {
  assertExecutionEqual,
  stateManagerAddress,
  whitelistedOpcodes,
} from '../helpers'
import { EvmIntrospectionUtil } from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'

/**
 * Validates transpiled JUMP bytecode provided via the TranspilationResult parameter.
 *
 * @param successResult The transpilation result in question.
 */
const validateJumpBytecode = (successResult: SuccessfulTranspilation): void => {
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

  const switchSuccessJumpdestIndex: number = jumpdestIndexes.pop()
  const switchSuccessJumpdest: Buffer = successResult.bytecode.slice(
    switchSuccessJumpdestIndex,
    switchSuccessJumpdestIndex + 1
  )
  switchSuccessJumpdest.should.eql(
    Opcode.JUMPDEST.code,
    `Switch success JUMPDEST index is ${switchJumpdestIndex}, but byte at that index is ${bufToHexString(
      switchJumpdest
    )}, not ${bufToHexString(Opcode.JUMPDEST.code)}`
  )

  opcodesBeforeJump.size.should.be.greaterThan(
    0,
    'opcodesBeforeJump should have entries but does not!'
  )

  for (const [index, opcodeBeforeJump] of opcodesBeforeJump.entries()) {
    if (index < switchSuccessJumpdestIndex) {
      // All regular program JUMPs should go to the footer JUMPDEST
      opcodeBeforeJump.opcode.programBytesConsumed.should.be.gt(
        0,
        'Opcode before JUMP should be a PUSH32, pushing the location of the footer JUMP switch!'
      )
      opcodeBeforeJump.consumedBytes.should.eql(
        bufferUtils.numberToBuffer(switchJumpdestIndex),
        'JUMP should be equal to index of footer switch JUMPDEST!'
      )
    } else if (index > switchJumpdestIndex) {
      // Make sure that all footer JUMPS go to footer JUMP success jumpdest
      const dest: number = opcodeBeforeJump.consumedBytes.readInt32BE(28)
      dest.should.eq(
        switchSuccessJumpdestIndex,
        'All footer JUMPs should go to success JUMPDEST block'
      )
    }
  }
}

const getSuccessfulTranspilationResult = (
  transpiler: Transpiler,
  bytecode: Buffer
): SuccessfulTranspilation => {
  const result: TranspilationResult = transpiler.transpile(bytecode)
  result.succeeded.should.equal(true)
  return result as SuccessfulTranspilation
}

describe('Transpile - JUMPs', () => {
  let opcodeWhitelist: OpcodeWhitelist
  let transpiler: Transpiler
  let replacer: OpcodeReplacer
  let evmUtil: EvmIntrospectionUtil

  beforeEach(async () => {
    opcodeWhitelist = new OpcodeWhitelistImpl(whitelistedOpcodes)
    replacer = new OpcodeReplacerImpl(
      stateManagerAddress,
      new Map<EVMOpcode, EVMBytecode>()
    )
    transpiler = new TranspilerImpl(opcodeWhitelist, replacer)
    evmUtil = await EvmIntrospectionUtilImpl.create()
  })

  it('handles simple JUMPs properly', async () => {
    const evmBytecode: EVMBytecode = [
      { opcode: Opcode.PUSH32, consumedBytes: bufferUtils.numberToBuffer(34) },
      { opcode: Opcode.JUMP, consumedBytes: undefined },
      { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
      { opcode: Opcode.STOP, consumedBytes: undefined },
    ]
    const initialBytecode: Buffer = bytecodeToBuffer(evmBytecode)

    const successResult: SuccessfulTranspilation = getSuccessfulTranspilationResult(
      transpiler,
      initialBytecode
    )

    validateJumpBytecode(successResult)
    await assertExecutionEqual(evmUtil, initialBytecode, successResult.bytecode)
  })

  it('handles simple JUMPIs properly', async () => {
    const evmBytecode: EVMBytecode = [
      { opcode: Opcode.PUSH32, consumedBytes: bufferUtils.numberToBuffer(1) },
      {
        opcode: Opcode.PUSH32,
        consumedBytes: bufferUtils.numberToBuffer(67),
      },
      { opcode: Opcode.JUMPI, consumedBytes: undefined },
      { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
      { opcode: Opcode.STOP, consumedBytes: undefined },
    ]
    const initialBytecode: Buffer = bytecodeToBuffer(evmBytecode)

    const successResult: SuccessfulTranspilation = getSuccessfulTranspilationResult(
      transpiler,
      initialBytecode
    )
    validateJumpBytecode(successResult)
    await assertExecutionEqual(evmUtil, initialBytecode, successResult.bytecode)
  })

  it('handles complex JUMP(I)s properly', async () => {
    const evmBytecode: EVMBytecode = [
      { opcode: Opcode.PUSH32, consumedBytes: bufferUtils.numberToBuffer(1) },
      {
        opcode: Opcode.PUSH32,
        consumedBytes: bufferUtils.numberToBuffer(103),
      },
      { opcode: Opcode.JUMPI, consumedBytes: undefined },
      { opcode: Opcode.PUSH1, consumedBytes: bufferUtils.numberToBuffer(1) },
      { opcode: Opcode.DUP1, consumedBytes: undefined },
      { opcode: Opcode.SWAP1, consumedBytes: undefined },
      { opcode: Opcode.DIV, consumedBytes: undefined },
      { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
      { opcode: Opcode.STOP, consumedBytes: undefined },
      { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
      { opcode: Opcode.RETURN, consumedBytes: undefined },
      {
        opcode: Opcode.PUSH32,
        consumedBytes: bufferUtils.numberToBuffer(107),
      },
      { opcode: Opcode.JUMP, consumedBytes: bufferUtils.numberToBuffer(1) },
    ]
    const initialBytecode: Buffer = bytecodeToBuffer(evmBytecode)

    const successResult: SuccessfulTranspilation = getSuccessfulTranspilationResult(
      transpiler,
      initialBytecode
    )
    validateJumpBytecode(successResult)
    await assertExecutionEqual(evmUtil, initialBytecode, successResult.bytecode)
  })

  it('handles code without JUMPs properly', async () => {
    const evmBytecode: EVMBytecode = [
      {
        opcode: Opcode.PUSH32,
        consumedBytes: bufferUtils.numberToBuffer(67),
      },
      { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
      { opcode: Opcode.STOP, consumedBytes: undefined },
    ]
    const initialBytecode: Buffer = bytecodeToBuffer(evmBytecode)

    const successResult: SuccessfulTranspilation = getSuccessfulTranspilationResult(
      transpiler,
      initialBytecode
    )
    validateJumpBytecode(successResult)
    await assertExecutionEqual(evmUtil, initialBytecode, successResult.bytecode)
  })

  it('handles code without JUMPs or JUMPDESTs properly', async () => {
    const evmBytecode: EVMBytecode = [
      {
        opcode: Opcode.PUSH32,
        consumedBytes: bufferUtils.numberToBuffer(67),
      },
    ]
    const initialBytecode: Buffer = bytecodeToBuffer(evmBytecode)

    const result: TranspilationResult = transpiler.transpile(initialBytecode)
    result.succeeded.should.equal(
      true,
      'Transpilation should have succeeded but did not!'
    )

    const successResult: SuccessfulTranspilation = result as SuccessfulTranspilation

    successResult.bytecode.should.eql(
      initialBytecode,
      'Transpilation should not have changed anything but did!'
    )

    await assertExecutionEqual(evmUtil, initialBytecode, successResult.bytecode)
  })
})
