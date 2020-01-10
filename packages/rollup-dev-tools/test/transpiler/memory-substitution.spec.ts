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
  pushMemoryOntoStack,
  storeStackInMemory,
  pushMemoryOntoStackAtIndex,
  storeStackInMemoryAtIndex,
  getPUSHIntegerOp,
} from '../../src/tools/transpiler'
import { stateManagerAddress, whitelistedOpcodes } from '../helpers'
import { EvmIntrospectionUtil, StepContext } from '../../src/types/vm'
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

  describe('Memory/stack swapping', () => {
    it('Correctly pushes multiple words of memory into the stack', async () => {
      const numWords: number = 3
      const byteIndexToLoad: number = 3
      const storeAndPushToStack: EVMBytecode = [
        ...storeNWordsInMemorySequential(9), // random exceeding numwords + index
        ...pushMemoryOntoStackAtIndex(byteIndexToLoad, numWords),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const finalStep: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(storeAndPushToStack),
        624 // hardcoded based on above vars -- changing them will require updating this
      )
      log.debug(`Final step context was: ${JSON.stringify(finalStep)}`)
      // The stack should only contain the loaded words
      finalStep.stackDepth.should.equal(numWords)
      // The stack should contain the loaded words in reverse order
      for (let i = 0; i < numWords; i++) {
        const expectedWordStart: number = byteIndexToLoad + 32 * i
        const expectedMemorySlice: Buffer = Buffer.from(
          finalStep.memory.slice(expectedWordStart, expectedWordStart + 32)
        )
        const expectedStackIndex: number = numWords - i - 1 // they're pushed onto stack in reverse order
        const wordOnStack: Buffer = finalStep.stack[expectedStackIndex]
        // check equality, etherjs-vm removes unneceessary zeroes so compare numerically
        new BigNumber(expectedMemorySlice).eq(new BigNumber(wordOnStack)).should
          .be.true
      }
    })
    it('Correctly stores multiple words from the stack back into memory', async () => {
      const fourRandomWords: Buffer = hexStrToBuf(
        '0x0111030ffffa0a0a11103040a0a0a0a011103040a0a0a0a1110232323a0a0a0d011103040a0a0a0a11103040a555555011103040a0a0a0a11103040a0a0a0ab0111030ffffa0a0a11103040a0a0a0a011103040adddddd1110232323a0a0a07011103040a0a0a5858699040a555555011103040a0a0a0a11103040a0a0a0abbb'
      )
      const pushWordsToStack: EVMBytecode = []
      for (let i = 0; i < 4; i++) {
        pushWordsToStack.push({
          opcode: Opcode.PUSH32,
          consumedBytes: fourRandomWords.slice(i * 32, (i + 1) * 32),
        })
      }
      const pushWordsToStackAndRestore: EVMBytecode = [
        ...pushWordsToStack,
        ...storeStackInMemoryAtIndex(0, 4),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const finalStep: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(pushWordsToStackAndRestore),
        159 // hardcoded based on above vars -- changing them will require updating this
      )
      finalStep.stackDepth.should.equal(0)
      finalStep.memory.should.deep.equal(fourRandomWords)
    })
    it('Memory operations between a pushtoStack and storeInMemory operation should not have any effect', async () => {
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
      // push memory to stack, overwrite memory, store stack back to memory
      const pusModifyLoad: EVMBytecode = [
        ...pushMemoryOntoStackAtIndex(memoryIndexToModify, numWordsToModify),
        ...overwriteNWordsInMemoryWithOffset(
          numWordsToModify,
          memoryIndexToModify
        ),
        ...storeStackInMemoryAtIndex(memoryIndexToModify, numWordsToModify),
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
        ...pusModifyLoad,
      ])

      const opcodeWhitelist = new OpcodeWhitelistImpl(whitelistedOpcodes)
      const replacer = new OpcodeReplacerImpl(stateManagerAddress, replaceMap)
      const transpiler = new TranspilerImpl(opcodeWhitelist, replacer)
      const transpilation = transpiler.transpile(
        memoryModifyingBytecodeBuf
      ) as SuccessfulTranspilation
      const transpiledMemoryModifyingBytecodeBuf: Buffer =
        transpilation.bytecode

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
        775
      )

      comparisonBeforeReturns.firstContext.memory.should.deep.equal(
        comparisonBeforeReturns.secondContext.memory
      )
      comparisonBeforeReturns.firstContext.stack.should.deep.equal(
        comparisonBeforeReturns.secondContext.stack
      )
    })
  })
})
