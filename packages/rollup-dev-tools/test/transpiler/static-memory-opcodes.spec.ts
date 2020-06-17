/* External Imports */
import { ethers } from 'ethers'
import {
  bufToHexString,
  remove0x,
  getLogger,
  hexStrToBuf,
  bufferUtils,
} from '@eth-optimism/core-utils'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  formatBytecode,
  Opcode,
  EVMOpcodeAndBytes,
} from '@eth-optimism/rollup-core'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Internal Imports */
import {
  EvmIntrospectionUtil,
  ExecutionResult,
  StepContext,
  CallContext,
  EvmIntrospectionUtilImpl,
  getPUSHBuffer,
  getPUSHIntegerOp,
  duplicateStackAt,
  callContractWithStackElementsAndReturnWordToMemory,
  storeStackElementsAsMemoryWords,
  callContractWithStackElementsAndReturnWordToStack,
} from '../../src'

const log = getLogger(`test-static-memory-opcodes`, true)
const abi = new ethers.utils.AbiCoder()

/* Contracts */
import * as AssemblyReturnGetter from '../contracts/build/AssemblyReturnGetter.json'

describe('Static Memory Opcode Replacement', () => {
  let evmUtil: EvmIntrospectionUtil
  const contractBytecode: Buffer = Buffer.from(
    AssemblyReturnGetter.bytecode,
    'hex'
  )
  const getterFunctionName: string = 'get'
  const getterMethodId: Buffer = ethereumjsAbi.methodID(getterFunctionName, [])
  const valToReturn: Buffer = hexStrToBuf(
    '0xbeadfeedbeadfeedbeadfeedbeadfeedbeadfeedbeadfeedbeadfeedbeadfeed'
  )
  const contractDeployParams: Buffer = Buffer.from(
    remove0x(abi.encode(['bytes'], [bufToHexString(valToReturn)])),
    'hex'
  )

  let getterAddress: Address

  const deployAssemblyReturningContract = async (
    util: EvmIntrospectionUtil
  ): Promise<Address> => {
    const result: ExecutionResult = await util.deployContract(
      contractBytecode,
      contractDeployParams
    )
    return bufToHexString(result.result)
  }
  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
    // deploy the contract whose function `get` returns raw non ABI-encoded bytes
    getterAddress = await deployAssemblyReturningContract(evmUtil)
  })
  describe('Some helpers', () => {
    it('Should correctly duplicateStackAt', async () => {
      const initialStackSize: number = 7
      const offsetToDuplicate: number = 3
      const elementsToDuplicate: number = 2

      const op: EVMBytecode = [
        ...pushStackElements(initialStackSize),
        ...duplicateStackAt(offsetToDuplicate, elementsToDuplicate),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(op),
        233
      )

      finalContext.stackDepth.should.equal(
        initialStackSize + elementsToDuplicate
      )
      const finalStack: Buffer[] = finalContext.stack
      finalStack
        .slice(0, elementsToDuplicate)
        .should.deep.equal(
          finalStack.slice(
            elementsToDuplicate + offsetToDuplicate,
            offsetToDuplicate + 2 * elementsToDuplicate
          )
        )
    })
  })

  describe('storeStackElementsAsMemoryWords', () => {
    it('Should store three stack elements in the memory, including the first stack element', async () => {
      const stackElements: Buffer[] = [
        Buffer.alloc(32).fill(hexStrToBuf('0xaa')),
        Buffer.alloc(32).fill(hexStrToBuf('0xbb')),
        Buffer.alloc(32).fill(hexStrToBuf('0xcc')),
      ]
      // bytecode to push the `stackElements` array to the stack, done in reverse order so stack is [aa, bb, cc]
      const pushAndStore: EVMBytecode = [
        getPUSHBuffer(stackElements[2]),
        getPUSHBuffer(stackElements[1]),
        getPUSHBuffer(stackElements[0]),
        ...storeStackElementsAsMemoryWords(3),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      log.debug(
        `running the following storeStackElementsAsMemoryWords bytecode: \n${formatBytecode(
          pushAndStore
        )}`
      )
      const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(pushAndStore),
        bytecodeToBuffer(pushAndStore).length - 1
      )
      // memory should be the concatenation of the 32 byte words we previously pushed to the stack
      finalContext.memory.should.deep.equal(Buffer.concat(stackElements))
    })
  })

  describe('callContractWithStackElementsAndReturnWordToMemory', () => {
    it('Should return the result of a simple contract getter with 0 stack params to memory successfully', async () => {
      // get bytecode which calls contract, passing stack elements, and returning the word to memory
      const callGetterAndStore: EVMBytecode = callContractWithStackElementsAndReturnWordToMemory(
        getterAddress,
        getterFunctionName,
        0
      )
      callGetterAndStore.push({
        opcode: Opcode.RETURN,
        consumedBytes: undefined,
      })

      const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(callGetterAndStore),
        73
      )

      const methodBytes: Buffer = ethereumjsAbi.methodID(getterFunctionName, [])
      // the resulting memory should be: [0000s proceeding method Id, methodId], [valToReturn]
      // where [] above indicates a 32 byte word.
      const expectedMemorySlice: Buffer = Buffer.concat([
        Buffer.alloc(32 - 4),
        methodBytes,
        valToReturn,
      ])

      finalContext.memory.should.deep.equal(expectedMemorySlice)
    })
    it('Should return the result of a simple contract getter with with stack params 0 and 1 to memory successfully', async () => {
      const numStackElementsToPass: number = 3

      let callGetterAndStoreWithStackParams: EVMBytecode = callContractWithStackElementsAndReturnWordToMemory(
        getterAddress,
        getterFunctionName,
        numStackElementsToPass
      )

      callGetterAndStoreWithStackParams = [
        ...pushStackElements(numStackElementsToPass),
        ...callGetterAndStoreWithStackParams,
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      log.debug(
        `Running getter-storing bytecode which pushes elements to stack: \n${formatBytecode(
          callGetterAndStoreWithStackParams
        )}`
      )

      const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(callGetterAndStoreWithStackParams),
        181
      )
      // the resulting memory should be: [0000s proceeding method Id, methodId], [stack param 0], [stack param 1], [stack param 2] [deadbeef, 0000...]
      // where [] above indicates a 32 byte word.
      const expectedMemorySlice: Buffer = hexStrToBuf(
        '0x000000000000000000000000000000000000000000000000000000006d4ce63c000000000000000000000000000000000000000000000000000000000000000001010101010101010101010101010101010101010101010101010101010101010202020202020202020202020202020202020202020202020202020202020202' +
          remove0x(bufToHexString(valToReturn))
      )

      finalContext.memory.should.deep.equal(expectedMemorySlice)
    })
  })
  describe('callContractWithStackElementsAndReturnWordToStack', () => {
    const initialMemory: Buffer = Buffer.alloc(32 * 10).fill(25)
    const aBigStack: Buffer[] = Array.from({ length: 10 }, (v, k) =>
      Buffer.from(new Array<number>(32).fill(k))
    ) // this whole thing gets us [0x0000, 0x010101, 0x020202, ...] as 32 byte words

    for (const numStackElsToPass of [0, 1, 2]) {
      for (const numWordsToReturn of [0, 1]) {
        const thisStack: Buffer[] = aBigStack.slice(0, numStackElsToPass + 1) // +1 since first stack element preserved
        it(`Should successfully pass ${numStackElsToPass} concatenated stack elements and methodId as calldata and return ${numWordsToReturn} words to the stack`, async () => {
          const setupContextAndExecuteCall: EVMBytecode = setupAndExecuteStaticMemoryCall(
            getterAddress,
            getterFunctionName,
            initialMemory,
            thisStack,
            numWordsToReturn as 0 | 1
          )

          log.debug(
            `setupAndExecuteStaticMemoryCall(...) bytecode for ${numStackElsToPass} numStackElsToPass and ${numWordsToReturn} numWordsToReturn is: \n${formatBytecode(
              setupContextAndExecuteCall
            )}`
          )

          const callContext: CallContext = await evmUtil.getCallContext(
            bytecodeToBuffer(setupContextAndExecuteCall)
          )
          // make sure the calldata is [methodId, thisStack[1], thisStack[2], ...]
          callContext.callData.should.deep.equal(
            Buffer.concat([getterMethodId, ...thisStack.slice(1)]),
            'Calldata should always be [bytes4 methodId], [stack el 1], [stack el 2]'
          )

          log.debug(`getting final cntexts`)

          const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
            bytecodeToBuffer(setupContextAndExecuteCall),
            bytecodeToBuffer(setupContextAndExecuteCall).length - 1
          )
          // make sure the memory was not at all affected
          finalContext.memory.should.deep.equal(
            initialMemory,
            'Memory should not change over the course of memory-static opcode replacements.'
          )
          // make sure the returnData was pushed to stack if needed
          finalContext.stackDepth.should.equal(
            numWordsToReturn + 1,
            `Stack does not match requested number of words returned(${numWordsToReturn}) + 1 for preserved first stack element`
          )
          bufferUtils
            .padLeft(finalContext.stack[0], 32)
            .should.deep.eq(
              thisStack[0],
              'Operation did not preserve the first stack element correctly'
            )
          if (numWordsToReturn === 1) {
            finalContext.stack[1].should.deep.equal(
              valToReturn,
              'Word returned to stack was not what the getter was told to return!'
            )
          }
        })
      }
    }
  })
})

// helper function to generate bytecode which:
// 1. Fills memory as requested
// 2. Sets up the stack as requested
// 3. Executes a setupContextCALLandReturnBuf
// 4. Returns
const setupAndExecuteStaticMemoryCall = (
  callTarget: Address,
  targetMethodName: string,
  preOperationMemory: Buffer,
  initialStack: Buffer[],
  numWordsToBeReturned: 0 | 1
) => {
  const replacementOperation: EVMBytecode = callContractWithStackElementsAndReturnWordToStack(
    callTarget,
    targetMethodName,
    initialStack.length - 1, // since we are not passing the first stack element
    numWordsToBeReturned
  )
  // push to stack in  reverse order so that we stack[0] is pushed last
  const setStack: EVMBytecode = initialStack
    .slice() // slice so we don't reverse original array (reused in testing)
    .reverse()
    .map((stackEl: Buffer): EVMOpcodeAndBytes => getPUSHBuffer(stackEl))
  return [
    ...setMemory(preOperationMemory),
    ...setStack,
    ...replacementOperation,
    {
      opcode: Opcode.RETURN,
      consumedBytes: undefined,
    },
  ]
}

// Helper function, sets the memory to the given buffer
const setMemory = (toSet: Buffer): EVMBytecode => {
  let op: EVMBytecode = []
  const numWords = Math.ceil(toSet.byteLength / 32)
  for (let i = 0; i < numWords; i++) {
    op = op.concat([
      getPUSHBuffer(toSet.slice(i * 32, (i + 1) * 32)),
      getPUSHIntegerOp(i * 32),
      {
        opcode: Opcode.MSTORE,
        consumedBytes: undefined,
      },
    ])
  }
  return op
}

// helper function, sets stack to [00 00 00 00 ...] [01 01 01 01 ...] ... [0N 0N 0N ...]
const pushStackElements = (numElements: number): EVMBytecode => {
  const op: EVMBytecode = []
  for (let i = 0; i < numElements; i++) {
    op.push(getPUSHBuffer(Buffer.alloc(32).fill(numElements - i - 1)))
  }
  return op
}
