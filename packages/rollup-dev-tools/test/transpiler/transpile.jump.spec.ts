import { should } from '../setup'

/* External Imports */
import { bufferUtils, bufToHexString } from '@eth-optimism/core-utils'
import {
  Opcode,
  EVMOpcode,
  EVMBytecode,
  bytecodeToBuffer,
  bufferToBytecode,
  EVMOpcodeAndBytes,
  formatBytecode,
} from '@eth-optimism/rollup-core'

/* Internal imports */
import {
  ErroredTranspilation,
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

const getSuccessfulTranspilationResult = (
  transpiler: Transpiler,
  bytecode: Buffer
): SuccessfulTranspilation => {
  const result: TranspilationResult = transpiler.transpileRawBytecode(bytecode)
  result.succeeded.should.equal(
    true,
    `${JSON.stringify((result as ErroredTranspilation).errors)}`
  )
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
      {
        opcode: Opcode.PUSH2,
        consumedBytes: bufferUtils.numberToBufferPacked(4, 2),
      },
      { opcode: Opcode.JUMP, consumedBytes: undefined },
      { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
      { opcode: Opcode.STOP, consumedBytes: undefined },
    ]
    const initialBytecode: Buffer = bytecodeToBuffer(evmBytecode)

    const successResult: SuccessfulTranspilation = getSuccessfulTranspilationResult(
      transpiler,
      initialBytecode
    )

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

    const result: TranspilationResult = transpiler.transpileRawBytecode(
      initialBytecode
    )
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
