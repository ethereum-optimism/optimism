/* External Imports */
import { ethers } from 'ethers'
import {
  bufToHexString,
  remove0x,
  getLogger,
  hexStrToBuf,
  bufferUtils,
} from '@pigi/core-utils'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  formatBytecode,
  Opcode,
} from '@pigi/rollup-core'

/* Internal Imports */
import {
  EvmIntrospectionUtil,
  ExecutionResult,
  StepContext,
} from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import {
  duplicateStackAt,
  callContractWithStackElementsAndReturnWordToMemory,
  storeStackElementsAsMemoryWords,
  callContractWithStackElementsAndReturnWordToStack,
} from '../../src/tools/transpiler/static-memory-opcodes'

const log = getLogger(`test-static-memory-opcodes`)

import * as abiForMethod from 'ethereumjs-abi'
const abi = new ethers.utils.AbiCoder()

/* Contracts */
import * as AssemblyReturnGetter from '../contracts/build/AssemblyReturnGetter.json'
import {
  getPUSHBuffer,
  getPUSHIntegerOp,
} from '../../src/tools/transpiler/memory-substitution'

describe('Static Memory Opcode Replacement', () => {
  let evmUtil: EvmIntrospectionUtil
  const contractBytecode: Buffer = Buffer.from(
    AssemblyReturnGetter.bytecode,
    'hex'
  )
  const getterFunctionName: string = 'get'
  const contractDeployParams: Buffer = Buffer.from(
    remove0x(abi.encode(['bytes'], ['0xdeadbeef'])),
    'hex'
  )

  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
  })

  const deployAssemblyReturningContract = async (
    util: EvmIntrospectionUtil
  ): Promise<Address> => {
    const result: ExecutionResult = await util.deployContract(
      contractBytecode,
      contractDeployParams
    )
    return bufToHexString(result.result)
  }
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
    it('Should store three stack elements in the memory', async () => {
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
        108
      )
      // memory should be the concatenation of the 32 byte words we previously pushed to the stack
      finalContext.memory.should.deep.equal(Buffer.concat(stackElements))
    })
  })

  describe('callContractWithStackElementsAndReturnWordToMemory', () => {
    it('Should return the result of a simple contract getter with 0 stack params to memory successfully', async () => {
      // deploy the contract whose function `get` returns raw non ABI-encoded bytes
      const getterAddress: Address = await deployAssemblyReturningContract(
        evmUtil
      )
      // get bytecode which calls contract, passing stack elements, and returning the word to memory
      let callGetterAndStore: EVMBytecode = callContractWithStackElementsAndReturnWordToMemory(
        getterAddress,
        getterFunctionName,
        0
      )

      callGetterAndStore = [
        ...callGetterAndStore,
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      log.debug(
        `Running getter-storing bytecode: \n${formatBytecode(
          callGetterAndStore
        )}`
      )

      const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(callGetterAndStore),
        73
      )

      const methodBytes: Buffer = abiForMethod.methodID(getterFunctionName, [])
      // the resulting memory should be: [0000s proceeding method Id, methodId], [deadbeef, 0000...]
      // where [] above indicates a 32 byte word.
      const expectedMemorySlice: Buffer = Buffer.concat([
        Buffer.alloc(32 - 4),
        methodBytes,
        Buffer.from('deadbeef', 'hex'),
        Buffer.alloc(32 - 4),
      ])

      finalContext.memory.should.deep.equal(expectedMemorySlice)
    })
    it('Should return the result of a simple contract getter with 2 stack params to memory successfully', async () => {
      const getterAddress: Address = await deployAssemblyReturningContract(
        evmUtil
      )

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
      // the resulting memory should be: [0000s proceeding method Id, methodId], [stack param 1], [stack param 2], [stack param 3] [deadbeef, 0000...]
      // where [] above indicates a 32 byte word.
      const expectedMemorySlice: Buffer = hexStrToBuf(
        '0x000000000000000000000000000000000000000000000000000000006d4ce63c000000000000000000000000000000000000000000000000000000000000000001010101010101010101010101010101010101010101010101010101010101010202020202020202020202020202020202020202020202020202020202020202deadbeef00000000000000000000000000000000000000000000000000000000'
      )

      finalContext.memory.should.deep.equal(expectedMemorySlice)
    })
  })
  describe('callContractWithStackElementsAndReturnWordToStack', () => {
    it('Should return the result of a simple contract getter with 2 stack params to memory successfully', async () => {
      const getterAddress: Address = await deployAssemblyReturningContract(
        evmUtil
      )

      const numStackElementsToPass: number = 3

      let callGetterAndStoreWithStackParams: EVMBytecode
      callGetterAndStoreWithStackParams = callContractWithStackElementsAndReturnWordToStack(
        getterAddress,
        getterFunctionName,
        numStackElementsToPass
      )

      const initialMemory: Buffer = Buffer.alloc(32 * 10).fill(25)

      callGetterAndStoreWithStackParams = [
        // fill memory with some data so that we can confirm it was not modified
        ...setMemory(initialMemory),
        // push the stack elements we're simulating passing in parameters
        ...pushStackElements(numStackElementsToPass),
        // execute the call which should not modify memory but get the return val into the stack
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
        661
      )
      // makesurethat memory was not modified
      finalContext.memory.should.deep.equal(initialMemory)
      // make sure deadbeef was put on  the stack
      finalContext.stackDepth.should.equal(1)
      finalContext.stack[0]
        .slice(0, 4)
        .should.deep.equal(hexStrToBuf('0xdeadbeef'))
    })
  })
})

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
