import { should } from '../setup'

/* External Imports */
import {
  getLogger,
  Logger,
  hexStrToBuf,
  BigNumber,
  bufferUtils,
} from '@pigi/core-utils'
import {
  Opcode,
  EVMOpcode,
  EVMBytecode,
  bytecodeToBuffer,
  bufferToBytecode,
  formatBytecode,
} from '@pigi/rollup-core'

/* Internal imports */
import { SuccessfulTranspilation } from '../../src/types/transpiler'
import {
  TranspilerImpl,
  OpcodeReplacerImpl,
  OpcodeWhitelistImpl,
  pushMemoryAtIndexOntoStack,
  storeStackInMemoryAtIndex,
  getPUSHIntegerOp,
} from '../../src/tools/transpiler'
import { stateManagerAddress, whitelistedOpcodes } from '../helpers'
import { EvmIntrospectionUtil, StepContext } from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'

const log: Logger = getLogger('test-memory-sub')

const overwritingString: string = '69' // nice.
const overwritingByte: Buffer = Buffer.from(overwritingString, 'hex')
const overwritingBytes32: Buffer = Buffer.from(
  overwritingString.repeat(32),
  'hex'
)

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
  const storageBytecode: EVMBytecode[] = []
  for (let i = 0; i < numWords; i++) {
    storageBytecode.push([
      {
        opcode: Opcode.PUSH32,
        consumedBytes: Buffer.alloc(32).fill(i),
      },
      {
        opcode: Opcode.PUSH32,
        consumedBytes: bufferUtils.numberToBuffer(i * 32),
      },
      {
        opcode: Opcode.MSTORE,
        consumedBytes: undefined,
      },
    ])
  }
  return [].concat(...storageBytecode)
}

const getExpectedMemoryAfterSequentialStore = (numWords): Buffer => {
  const expectedMemory: Buffer = Buffer.alloc(numWords * 32)
  for (let i = 0; i < numWords; i++) {
    expectedMemory.fill(i, i * 32, (i + 1) * 32)
  }
  return expectedMemory
}

const overwriteNWordsInMemoryWithOffset = (
  numWords: number,
  offset: number
): EVMBytecode => {
  const overwriteBytecode: EVMBytecode[] = []
  for (let i = 0; i < numWords; i++) {
    overwriteBytecode.push([
      {
        opcode: Opcode.PUSH32,
        consumedBytes: overwritingBytes32,
      },
      getPUSHIntegerOp(offset + i * 32),
      {
        opcode: Opcode.MSTORE,
        consumedBytes: undefined,
      },
    ])
  }
  return [].concat(...overwriteBytecode)
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

      const indexOfReturnOp: number = operationBuffer.length - 1
      const memoryStoredResult: StepContext = await evmUtil.getStepContextBeforeStep(
        operationBuffer,
        indexOfReturnOp
      )
      memoryStoredResult.stackDepth.should.equal(0)
      memoryStoredResult.memoryWordCount.should.equal(numSequentialWordsToStore)
      memoryStoredResult.memory.should.eql(
        getExpectedMemoryAfterSequentialStore(numSequentialWordsToStore)
      )
    })

    it('should correctly overwriteNWordsInMemoryWithOffset', async () => {
      const wordsToStore: number = 10
      const wordsToOverwrite: number = 3
      const overwriteOffset: number = 15
      const operationBytecode: EVMBytecode = [
        ...storeNWordsInMemorySequential(wordsToStore),
        ...overwriteNWordsInMemoryWithOffset(wordsToOverwrite, overwriteOffset),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const operationBuffer: Buffer = bytecodeToBuffer(operationBytecode)

      const indexOfReturnOp: number = operationBuffer.length - 1
      const memoryModifiedResult: StepContext = await evmUtil.getStepContextBeforeStep(
        operationBuffer,
        indexOfReturnOp
      )
      memoryModifiedResult.stackDepth.should.equal(0)
      memoryModifiedResult.memoryWordCount.should.equal(wordsToStore)

      const expectedMemory: Buffer = getExpectedMemoryAfterSequentialStore(
        wordsToStore
      )
      const bytesOverwritten = 32 * wordsToOverwrite
      expectedMemory.fill(
        overwritingByte,
        overwriteOffset,
        overwriteOffset + bytesOverwritten
      )

      memoryModifiedResult.memory.should.eql(expectedMemory)
    })
  })

  describe('Memory/stack swapping', () => {
    it('Correctly pushes multiple words of memory onto the stack', async () => {
      const numWords: number = 3
      const mstoreIndex: number = 3
      const storeAndPushToStack: EVMBytecode = [
        ...storeNWordsInMemorySequential(9), // random exceeding numWords + index
        ...pushMemoryAtIndexOntoStack(mstoreIndex, numWords),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const binary: Buffer = bytecodeToBuffer(storeAndPushToStack)
      const indexOfReturnOp: number = binary.length - 1
      const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
        binary,
        indexOfReturnOp
      )
      log.info(`Final step context was: ${JSON.stringify(finalContext)}`)

      finalContext.stackDepth.should.equal(
        numWords,
        'The stack should only contain the loaded words'
      )
      // The stack should contain the loaded words in order
      for (let i = 0; i < numWords; i++) {
        const wordIndex: number = mstoreIndex + 32 * i
        const wordFromMemory: Buffer = finalContext.memory.slice(
          wordIndex,
          wordIndex + 32
        )

        const wordOnStack: Buffer = finalContext.stack[i]
        // check equality, ethereumjs-vm removes unnecessary zeroes so compare numerically
        wordFromMemory.should.eql(
          Buffer.alloc(32).fill(wordOnStack, 32 - wordOnStack.length, 32)
        )
      }
    })

    it('Correctly stores multiple words from the stack back into memory', async () => {
      const fourRandomWords: Buffer = hexStrToBuf(
        '0x0111030ffffa0a0a11103040a0a0a0a011103040a0a0a0a1110232323a0a0a0d011103040a0a0a0a11103040a555555011103040a0a0a0a11103040a0a0a0ab0111030ffffa0a0a11103040a0a0a0a011103040adddddd1110232323a0a0a07011103040a0a0a5858699040a555555011103040a0a0a0a11103040a0a0a0abbb'
      )
      const pushWordsToStack: EVMBytecode = []
      for (let i = 3; i >= 0; i--) {
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
      const binary: Buffer = bytecodeToBuffer(pushWordsToStackAndRestore)
      const indexOfReturnOp: number = binary.length - 1
      const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
        binary,
        indexOfReturnOp
      )
      finalContext.stackDepth.should.equal(0)
      finalContext.memory.should.deep.equal(fourRandomWords)
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
      const pushModifyLoad: EVMBytecode = [
        ...pushMemoryAtIndexOntoStack(memoryIndexToModify, numWordsToModify),
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
        ...pushModifyLoad,
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
