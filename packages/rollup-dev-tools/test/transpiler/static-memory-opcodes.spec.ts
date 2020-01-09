/* External Imports */
import { ethers } from 'ethers'
import {
  bufToHexString,
  isValidHexAddress,
  remove0x,
  getLogger,
  BigNumber,
  hexStrToBuf,
} from '@pigi/core-utils'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  formatBytecode,
  Opcode,
  bufferToBytecode,
} from '@pigi/rollup-core'

/* Internal Imports */
import { should } from '../setup'
import {
  EvmErrors,
  EvmIntrospectionUtil,
  ExecutionResult,
  StepContext,
} from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import {
  emptyBuffer,
  getBytecodeCallingContractMethod,
  invalidBytesConsumedBytecode,
} from '../helpers'
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
  let contractBytecode: Buffer = Buffer.from(
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
            { opcode: Opcode.RETURN, consumedBytes: undefined}
        ]
        log.debug(`op is ${formatBytecode(op)}`)
        const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
            bytecodeToBuffer(op),
            233
        )

        finalContext.stackDepth.should.equal(initialStackSize + elementsToDuplicate)
        const finalStack: Buffer[] = finalContext.stack
        finalStack.slice(0, elementsToDuplicate).should.deep.equal(finalStack.slice(elementsToDuplicate + offsetToDuplicate, offsetToDuplicate + 2 * elementsToDuplicate))
    })
  })

  describe('storeStackElementsAsMemoryWords', () => {
    it('Should store three stack elements in the memory', async () => {
      const memoryIndextoStoreAt: number = 0
      const stackElements: Buffer[] = [
        Buffer.from(
          'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
          'hex'
        ),
        Buffer.from(
          'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
          'hex'
        ),
        Buffer.from(
          'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
          'hex'
        ),
      ]
      const pushAndStore: EVMBytecode = [
        getPUSHBuffer(stackElements[0]),
        getPUSHBuffer(stackElements[1]),
        getPUSHBuffer(stackElements[2]),
        ...storeStackElementsAsMemoryWords(memoryIndextoStoreAt, 3),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]

      log.debug(
        `Running stack-storing bytecode: \n${formatBytecode(pushAndStore)}`
      )
      const finalContext: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(pushAndStore),
        108
      )

      finalContext.memory.should.deep.equal(Buffer.concat(stackElements))
    })
  })

  describe('callContractWithStackElementsAndReturnWordToMemory', () => {
    it('Should return the result of a simple contract getter with 0 stack params to memory successfully', async () => {
      const getterAddress: Address = await deployAssemblyReturningContract(
        evmUtil
      )

      const memoryIndexToUse: number = 0

      let callGetterAndStore: EVMBytecode
      let memoryBytesUsed: number // todo assert  this val correct
      ;[
        callGetterAndStore,
        memoryBytesUsed,
      ] = callContractWithStackElementsAndReturnWordToMemory(
        getterAddress,
        getterFunctionName,
        0,
        memoryIndexToUse
      )
      // methodId + 0 stack elements passed to call + 1 words * 32 bytes returned
      memoryBytesUsed.should.equal(4 + 32)

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

      const memoryIndexToUse: number = 0

      const numStackElementsToPass: number = 3

      let callGetterAndStoreWithStackParams: EVMBytecode
      let memoryBytesUsed: number // todo assert this val correct
      ;[
        callGetterAndStoreWithStackParams,
        memoryBytesUsed,
      ] = callContractWithStackElementsAndReturnWordToMemory(
        getterAddress,
        getterFunctionName,
        numStackElementsToPass,
        memoryIndexToUse
      )
      // methodId + 0 stack elements passed to call + numStackElementsToPass * 32 bytes words + 1 words * 32 bytes returned
      memoryBytesUsed.should.equal(4 + 32 * numStackElementsToPass + 32)

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

      const expectedMemorySlice: Buffer = hexStrToBuf(
        '0x000000000000000000000000000000000000000000000000000000006d4ce63c020202020202020202020202020202020202020202020202020202020202020201010101010101010101010101010101010101010101010101010101010101010000000000000000000000000000000000000000000000000000000000000000deadbeef00000000000000000000000000000000000000000000000000000000'
      )

      finalContext.memory.should.deep.equal(expectedMemorySlice)
    })
  })
  describe('callContractWithStackElementsAndReturnWordToStack', () => {
    it('Should return the result of a simple contract getter with 2 stack params to memory successfully', async () => {
      const getterAddress: Address = await deployAssemblyReturningContract(
        evmUtil
      )

      const initialMemory: Buffer = Buffer.alloc(32 * 10).fill(25)
      const memoryIndexToUse: number = 0
      const numStackElementsToPass: number = 3

      let callGetterAndStoreWithStackParams: EVMBytecode
      callGetterAndStoreWithStackParams = callContractWithStackElementsAndReturnWordToStack(
        getterAddress,
        getterFunctionName,
        numStackElementsToPass,
        memoryIndexToUse
      )

      callGetterAndStoreWithStackParams = [
        ...setMemory(initialMemory),
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
        661
      )
      finalContext.memory.should.deep.equal(initialMemory)
    })
  })
})

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

const pushStackElements = (numElements: number): EVMBytecode => {
    let op: EVMBytecode = []
    for (let i = 0; i < numElements; i++) {
        op.push(
            getPUSHBuffer(Buffer.alloc(32).fill(numElements - i - 1))
        )
    }
    return op
}