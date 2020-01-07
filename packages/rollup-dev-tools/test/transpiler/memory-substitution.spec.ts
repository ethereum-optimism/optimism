import { should } from '../setup'

/* External Imports */
import {
  getLogger,
  Logger,
  bufferUtils,
  bufToHexString,
  hexStrToBuf,
  BigNumber,
} from '@pigi/core-utils'
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
  dynamicStashMemoryInStack,
  dynamicUnstashMemoryFromStack,
  staticStashMemoryInStack,
  staticUnstashMemoryFromStack,
  getPUSHIntegerOp,
} from '../../src/tools/transpiler'
import { stateManagerAddress, whitelistedOpcodes } from '../helpers'
import { EvmIntrospectionUtil } from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'

const log: Logger = getLogger('test-memory-sub')

const pointlessOperation: EVMBytecode = [
  {
    opcode: Opcode.PUSH1,
    consumedBytes: hexStrToBuf('0xab'),
  },
  {
    opcode: Opcode.POP,
    consumedBytes: undefined,
  },
]

const storeNWordsInMemorySequential = (numWords: number): EVMBytecode => {
  let storageBytecode: EVMBytecode = []
  for (let i = 0; i < numWords; i++) {
    storageBytecode = storageBytecode.concat([
      {
        opcode: Opcode.PUSH32,
        consumedBytes: Buffer.alloc(32).fill(new BigNumber(i).toBuffer('B', 1)),
      },
      {
        opcode: Opcode.PUSH32,
        consumedBytes: new BigNumber(i * 32).toBuffer('B', 32),
      },
      {
        opcode: Opcode.MSTORE,
        consumedBytes: undefined,
      },
    ])
  }
  return storageBytecode
}

const overwriteNWordsInMemoryWithOffset = (
  numWords: number,
  offset: number
): EVMBytecode => {
  let overwriteBytecode: EVMBytecode = []
  for (let i = 0; i < numWords; i++) {
    overwriteBytecode = overwriteBytecode.concat([
      {
        opcode: Opcode.PUSH32,
        consumedBytes: hexStrToBuf(
          '0x6969696969696969696969696969696969696969696969696969696969696969'
        ), // nice
      },
      getPUSHIntegerOp(offset + i * 32),
      {
        opcode: Opcode.MSTORE,
        consumedBytes: undefined,
      },
    ])
  }
  return overwriteBytecode
}

describe('Memory Replacement Operations', () => {
  let evmUtil: EvmIntrospectionUtil
  before(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
  })

  describe('Testing Memory Utils', () => {
    it('should correctly storeNWordsInMemorySequential', async () => {
      const numSequentialWordsToStore: number = 10
      const operationBytecode: EVMBytecode = [
        ...storeNWordsInMemorySequential(numSequentialWordsToStore),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const operationBuffer: Buffer = bytecodeToBuffer(operationBytecode)

      const memoryStoredResult = await evmUtil.getStepContextBeforeStep(
        operationBuffer,
        670 // hardcoded PC val, found via debug log
      )
      memoryStoredResult.stackDepth.should.equal(0)
      memoryStoredResult.memoryWordCount.should.equal(numSequentialWordsToStore)

      let expectedMemory: number[] = []
      for (let i = 0; i < numSequentialWordsToStore; i++) {
        expectedMemory = expectedMemory.concat(new Array(32).fill(i, 0, 32))
      }
      memoryStoredResult.memory.should.deep.equal(Buffer.from(expectedMemory))
    })
    it('should correctly overwriteNWordsInMemoryWithOffset', async () => {
      const numSequentialWordsToStore: number = 10
      const numSequentialWordsToOverwrite: number = 3
      const byteOffsetToOverwrite: number = 15
      const operationBytecode: EVMBytecode = [
        ...storeNWordsInMemorySequential(numSequentialWordsToStore),
        ...overwriteNWordsInMemoryWithOffset(
          numSequentialWordsToOverwrite,
          byteOffsetToOverwrite
        ),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const operationBuffer: Buffer = bytecodeToBuffer(operationBytecode)

      const memoryModifiedResult = await evmUtil.getStepContextBeforeStep(
        operationBuffer,
        778 // hardcoded  PC val, found via debug log
      )
      memoryModifiedResult.stackDepth.should.equal(0)
      memoryModifiedResult.memoryWordCount.should.equal(
        numSequentialWordsToStore
      )

      let expectedMemory: number[] = []
      for (let i = 0; i < numSequentialWordsToStore; i++) {
        expectedMemory = expectedMemory.concat(new Array(32).fill(i, 0, 32))
      }
      const numBytesOverWritten = 32 * numSequentialWordsToOverwrite
      expectedMemory.splice(
        byteOffsetToOverwrite,
        numBytesOverWritten,
        ...new Array(numBytesOverWritten).fill(105)
      ) // 105 is 0x69 in decimal

      memoryModifiedResult.memory.should.deep.equal(Buffer.from(expectedMemory))
    })
  })

  it('Memory operations between a stash and unstash operation should not have any effect', async () => {
    const numWordsToStore = 10
    const memoryModifyingBytecode: EVMBytecode = [
      ...storeNWordsInMemorySequential(numWordsToStore),
      ...pointlessOperation, // to be transpiled
      { opcode: Opcode.RETURN, consumedBytes: undefined },
    ]
    const memoryModifyingBytecodeBuf: Buffer = bytecodeToBuffer(
      memoryModifyingBytecode
    )

    const memoryIndexToModify: number = 2
    const numWordsToModify: number = 2
    // stash memory to stack, overwrite memory, unstash memory
    const stashModifyUnstash: EVMBytecode = [
      ...staticStashMemoryInStack(memoryIndexToModify, numWordsToModify),
      ...overwriteNWordsInMemoryWithOffset(
        numWordsToModify,
        memoryIndexToModify
      ),
      ...staticUnstashMemoryFromStack(memoryIndexToModify, numWordsToModify),
    ]

    const replaceMap: Map<EVMOpcode, EVMBytecode> = new Map<
      EVMOpcode,
      EVMBytecode
    >().set(Opcode.POP, [
      {
        // retain the POP we will be replacing so that the PUSH POP still has no effect
        opcode: Opcode.POP,
        consumedBytes: undefined,
      },
      ...stashModifyUnstash,
    ])

    const opcodeWhitelist = new OpcodeWhitelistImpl(whitelistedOpcodes)
    const replacer = new OpcodeReplacerImpl(stateManagerAddress, replaceMap)
    const transpiler = new TranspilerImpl(opcodeWhitelist, replacer)
    const transpilation = transpiler.transpile(
      memoryModifyingBytecodeBuf
    ) as SuccessfulTranspilation
    const transpiledMemoryModifyingBytecodeBuf: Buffer = transpilation.bytecode

    log.debug(
      `The memory modifying untranspiled bytecode is as follows: \n${formatBytecode(
        memoryModifyingBytecode
      )}\nAnd the transpiled version is: \n${formatBytecode(
        bufferToBytecode(transpiledMemoryModifyingBytecodeBuf)
      )}`
    )

    const comparisonBeforeReturns = await evmUtil.getExecutionComparisonBeforeStep(
      memoryModifyingBytecodeBuf,
      673,
      transpiledMemoryModifyingBytecodeBuf,
      788
    )

    comparisonBeforeReturns.firstContext.memory.should.deep.equal(
      comparisonBeforeReturns.secondContext.memory
    )
    comparisonBeforeReturns.firstContext.stack.should.deep.equal(
      comparisonBeforeReturns.secondContext.stack
    )
  })
})
