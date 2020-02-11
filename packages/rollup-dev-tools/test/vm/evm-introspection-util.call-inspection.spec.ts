/* External Imports */
import { ethers } from 'ethers'
import {
  bufToHexString,
  remove0x,
  hexStrToBuf,
  TestUtils,
} from '@eth-optimism/core-utils'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  Opcode,
} from '@eth-optimism/rollup-core'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Internal Imports */
import { should } from '../setup'
import {
  EvmIntrospectionUtil,
  ExecutionResult,
  CallContext,
  InvalidCALLStackError,
} from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import { setMemory, setupStackAndCALL } from '../helpers'

import * as AssemblyReturnGetter from '../contracts/build/AssemblyReturnGetter.json'
const abi = new ethers.utils.AbiCoder()

const getRandomWordsWithMethodIdAtOffset = (
  methodName: string,
  offset: number,
  totalWords: number
): Buffer => {
  const totalSizeBytes: number = 32 * totalWords
  return Buffer.alloc(totalSizeBytes)
    .fill(25, 0, offset)
    .fill(ethereumjsAbi.methodID(methodName, []), offset, offset + 4)
    .fill(69, offset + 4)
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
        argOffset,
        7 // random val
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

      callContext.stepContext.memory.should.deep.equal(
        memoryToFill,
        'context should have the pre-RETURNed memory'
      )
      callContext.stepContext.opcode.should.equal(
        Opcode.CALL,
        'context should be a CALL'
      )
      callContext.input.argLength.should.equal(
        argLength,
        'argLength should be the same as provided to call'
      )
      callContext.input.argOffset.should.equal(
        argOffset,
        'argOffset should be the same as provided to call'
      )
      const methodId: Buffer = ethereumjsAbi.methodID(getterMethodName, [])
      callContext.callData.should.deep.equal(
        methodId,
        'calldata should be the methodId'
      )
      callContext.input.addr.should.deep.equal(
        returnerAddress,
        'CALL should be headed to the returner address specified'
      )
      callContext.input.retOffset.should.equal(
        retOffset,
        'retOffset should be the same as provided to call'
      )
      callContext.input.retLength.should.equal(
        retLength,
        'retLength should be the same as provided to call'
      )
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
