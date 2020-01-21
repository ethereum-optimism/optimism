/* External Imports */
import { ethers } from 'ethers'
import {
  bufToHexString,
  isValidHexAddress,
  remove0x,
  hexStrToBuf,
  TestUtils,
} from '@pigi/core-utils'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  Opcode,
} from '@pigi/rollup-core'

/* Internal Imports */
import { should } from '../setup'
import {
  EvmErrors,
  EvmIntrospectionUtil,
  ExecutionResult,
  CallContext,
  StepContext,
  InvalidCALLStackError,
} from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import {
  emptyBuffer,
  getBytecodeCallingContractMethod,
  invalidBytesConsumedBytecode,
  setMemory,
  setupStackAndCALL,
} from '../helpers'

import {
  getPUSHBuffer,
  getPUSHIntegerOp,
} from '../../src/tools/transpiler/memory-substitution'

import * as AssemblyReturnGetter from '../contracts/build/AssemblyReturnGetter.json'
const abi = new ethers.utils.AbiCoder()

import * as abiForMethod from 'ethereumjs-abi'
import { CALL_EXCEPTION } from 'ethers/errors'

const getRandomWordsWithMethodIdAtOffset = (
  methodName: string,
  offset: number
): Buffer => {
  const memoryBeforeMethodId: Buffer = Buffer.alloc(offset).fill(25)
  const methodId: Buffer = abiForMethod.methodID(methodName, [])
  const totalSize: number = 32 * 7
  const memoryAfterMethodId: Buffer = Buffer.alloc(totalSize - offset - 4).fill(
    69
  )
  return Buffer.concat([memoryBeforeMethodId, methodId, memoryAfterMethodId])
}

const valToReturn: Buffer = hexStrToBuf('0xdeadbeef')
const deployGetterContract = async (
  util: EvmIntrospectionUtil
): Promise<Address> => {
  const getterBytecode: Buffer = Buffer.from(
    AssemblyReturnGetter.bytecode,
    'hex'
  )
  const result: ExecutionResult = await util.deployContract(
    getterBytecode,
    Buffer.from(remove0x(abi.encode(['bytes'], [valToReturn])), 'hex')
  )

  return bufToHexString(result.result)
}

const gas: number = 10001
const value: number = 0
const argOffset: number = 38
const argLength: number = 4
const retOffset: number = 7
const retLength: number = 4

describe('EvmIntrospectionUtil', () => {
  let evmUtil: EvmIntrospectionUtil
  let returnerAddress: Address
  const getterMethodName = 'get'

  describe('CallContext', () => {
    beforeEach(async () => {
      evmUtil = await EvmIntrospectionUtilImpl.create()
      returnerAddress = await deployGetterContract(evmUtil)
    })

    it('should successfully parse a CALL to a simple returner contract', async () => {
      const memoryToFill: Buffer = getRandomWordsWithMethodIdAtOffset(
        getterMethodName,
        argOffset
      )

      const fillMemoryAndCall: EVMBytecode = [
        ...setMemory(memoryToFill),
        ...setupStackAndCALL(
          gas,
          returnerAddress,
          value,
          argOffset,
          argLength,
          retOffset,
          retLength
        ),
        {
          opcode: Opcode.RETURN,
          consumedBytes: undefined,
        },
      ]

      const callContext: CallContext = await evmUtil.getCallContext(
        bytecodeToBuffer(fillMemoryAndCall)
      )

      // context should indeed be a CALL with the pre-RETURNed memory
      console.log(bufToHexString(callContext.stepContext.memory))
      callContext.stepContext.memory.should.deep.equal(memoryToFill)
      callContext.stepContext.opcode.should.equal(Opcode.CALL)
      // argsLocation should match up
      callContext.input.argLength.should.equal(argLength)
      callContext.input.argOffset.should.equal(argOffset)
      // calldata should be the methodId
      const methodId: Buffer = abiForMethod.methodID(getterMethodName, [])
      callContext.callData.should.deep.equal(methodId)
      // should be headed to the returner address specified
      callContext.input.addr.should.deep.equal(returnerAddress)
      // check return memory vals
      callContext.input.retOffset.should.equal(retOffset)
      callContext.input.retLength.should.equal(retLength)
    })

    it('Should pad calldata with 0s if exceeding memory size', async () => {
      const callWitOutOfBoundsMemory: EVMBytecode = setupStackAndCALL(
        gas,
        returnerAddress,
        value,
        1000, // since we set no memory beforehand this should be bigger than context.memory
        argLength,
        retOffset,
        retLength
      )
      const callContext: CallContext = await evmUtil.getCallContext(
        bytecodeToBuffer(callWitOutOfBoundsMemory)
      )
      callContext.callData.equals(Buffer.alloc(argLength)).should.be.true
    })

    it('Should throw if too few stack arguments', async () => {
      const callWithEmptyStack: EVMBytecode = [
        { opcode: Opcode.CALL, consumedBytes: undefined },
      ]
      TestUtils.assertThrowsAsync(async () => {
        await evmUtil.getCallContext(bytecodeToBuffer(callWithEmptyStack))
      }, InvalidCALLStackError)
    })
  })
})
