/* External Imports */
import { ethers } from 'ethers'
import {
  bufToHexString,
  remove0x,
  getLogger,
  hexStrToBuf,
  bufferUtils,
  hexStrToNumber,
} from '@eth-optimism/core-utils'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  Opcode,
  formatBytecode,
} from '@eth-optimism/rollup-core'
import * as abiForMethod from 'ethereumjs-abi'

/* Internal Imports */
import {
  EvmIntrospectionUtil,
  ExecutionResult,
  CallContext,
} from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import { setMemory, setupStackAndCALL } from '../helpers'
import {
  getCALLReplacement,
  getSTATICCALLReplacement,
  getEXTCODECOPYReplacement,
  BIG_ENOUGH_GAS_LIMIT,
} from '../../src'

const log = getLogger(`static-memory-opcodes`, true)

const abi = new ethers.utils.AbiCoder()

/* Contracts */
import * as AssemblyReturnGetter from '../contracts/build/AssemblyReturnGetter.json'
import { getPUSHBuffer, getPUSHIntegerOp } from '../../src/'

const valToReturn =
  '0xbeadfeedbeadfeedbeadfeedbeadfeedbeadfeedbeadfeedbeadfeedbeadfeed'
const contractDeployParams: Buffer = Buffer.from(
  remove0x(abi.encode(['bytes'], [valToReturn])),
  'hex'
)

describe('Memory-dynamic Opcode Replacement', () => {
  let evmUtil: EvmIntrospectionUtil
  const getMethodName: string = 'get'

  // mock up a CALL with random inputs
  const originalAddress: Address = '0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef'
  const retLength: number = 5
  const retoffset: number = 8 * 32
  const originalArgOffset: number = 4 * 32 + 17 // must exceed 4 * 32 for prepend to be possible
  const originalArgLength: number = 15
  const PCtoReturnTo: Buffer = hexStrToBuf('0x696969')

  const setupStackForCALL: EVMBytecode = setupStackAndCALL(
    BIG_ENOUGH_GAS_LIMIT,
    originalAddress,
    0,
    originalArgOffset,
    originalArgLength,
    retoffset,
    retLength
  )
  setupStackForCALL.pop() // pop the CALL itself
  setupStackForCALL.push(getPUSHBuffer(PCtoReturnTo)) // push a val to represent the PC to jump back to

  const deployGetterContract = async (
    util: EvmIntrospectionUtil
  ): Promise<Address> => {
    const contractBytecode: Buffer = Buffer.from(
      AssemblyReturnGetter.bytecode,
      'hex'
    )
    const result: ExecutionResult = await util.deployContract(
      contractBytecode,
      contractDeployParams
    )
    return bufToHexString(result.result)
  }

  let getterAddress: Address
  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
    getterAddress = await deployGetterContract(evmUtil)
    log.debug(`deployed getter contract to EVM introspection util.`)
  })

  describe('Call-type opcode replacements', () => {
    it.only('should parse a CALL replacement', async () => {
      // mock a transpiler-output replaced CALL
      const mockMemory: Buffer = Buffer.alloc(32 * 10).fill(25)
      const mockCallReplacement: EVMBytecode = [
        // fill memory with some random data so that we can confirm it was not modified
        ...setMemory(mockMemory),
        ...setupStackForCALL,
        ...getCALLReplacement(getterAddress, getMethodName),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]

      const callContext: CallContext = await evmUtil.getCallContext(
        bytecodeToBuffer(mockCallReplacement)
      )
      // check we generated the correct calldata
      const expectedCallData: Buffer = Buffer.concat([
        abiForMethod.methodID(getMethodName, []), // prepended methodId
        Buffer.alloc(32 - 20), // prepended address 32-byte word padding
        hexStrToBuf(originalAddress), // prepended Addreess
        mockMemory.slice(
          originalArgOffset,
          originalArgOffset + originalArgLength
        ), // original calldata
      ])

      callContext.callData.equals(expectedCallData).should.be.true

      // make sure the end state of memory is unaffectedx
      const finalContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(mockCallReplacement),
        bytecodeToBuffer(mockCallReplacement).length - 1
      )
      const expectedFinalMemory: Buffer = Buffer.concat([
        mockMemory.slice(0, retoffset),
        hexStrToBuf(valToReturn).slice(0, retLength),
        mockMemory.slice(retoffset + retLength),
      ])

      finalContext.memory.equals(expectedFinalMemory).should.be.true

      // check that the remaining stack is (PC to return to), (success)
      finalContext.stackDepth.should.equal(2)
      finalContext.stack[0].should.deep.equal(PCtoReturnTo)
      finalContext.stack[1].should.deep.equal(hexStrToBuf('0x01'))
    })
    it.only('should parse a STATICCALL replacement', async () => {
      // mock a transpiler-output replaced CALL
      const mockMemory: Buffer = Buffer.alloc(32 * 10).fill(25)
      // remove the VALUE param from the call
      setupStackForCALL.splice(4, 1)
      const mockCallReplacement: EVMBytecode = [
        // fill memory with some random data so that we can confirm it was not modified
        ...setMemory(mockMemory),
        ...setupStackForCALL,
        ...getSTATICCALLReplacement(getterAddress, getMethodName),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const callContext: CallContext = await evmUtil.getCallContext(
        bytecodeToBuffer(mockCallReplacement)
      )

      // check we generated the correct calldata
      const expectedCallData: Buffer = Buffer.concat([
        abiForMethod.methodID(getMethodName, []), // prepended methodId
        Buffer.alloc(32 - 20), // prepended address 32-byte word padding
        hexStrToBuf(originalAddress), // prepended Addreess
        mockMemory.slice(
          originalArgOffset,
          originalArgOffset + originalArgLength
        ), // original calldata
      ])

      callContext.callData.equals(expectedCallData).should.be.true

      // make sure the end state of memory is unaffectedx
      const finalContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(mockCallReplacement),
        bytecodeToBuffer(mockCallReplacement).length - 1
      )
      const expectedFinalMemory: Buffer = Buffer.concat([
        mockMemory.slice(0, retoffset),
        hexStrToBuf(valToReturn).slice(0, retLength),
        mockMemory.slice(retoffset + retLength),
      ])
      finalContext.memory.equals(expectedFinalMemory).should.be.true

      // check that the remaining stack is (PC to return to), (success)
      finalContext.stackDepth.should.equal(2)
      finalContext.stack[0].should.deep.equal(PCtoReturnTo)
      finalContext.stack[1].should.deep.equal(hexStrToBuf('0x01'))
    })
  })
  describe('EXTCODECOPY replacement', () => {
    const addressToRequest: Address =
      '0xbeeebeeebeeebeeebeeebeeebeeebeeeeeeeeeee'
    const length: number = 4
    const offset: number = 3
    const destOffset: number = 2
    const setupStackForEXTCODECOPY: EVMBytecode = [
      // fill memory with some random data so that we can confirm it was not modified
      ...setMemory(Buffer.alloc(32 * 10).fill(25)),
      getPUSHIntegerOp(length),
      getPUSHIntegerOp(offset),
      getPUSHIntegerOp(destOffset),
      getPUSHBuffer(hexStrToBuf(addressToRequest)), // address
    ]

    it('should correctly parse an EXTCODECOPY replacement', async () => {
      const extcodesizeReplacement: EVMBytecode = [
        ...setupStackForEXTCODECOPY,
        ...getEXTCODECOPYReplacement(getterAddress, getMethodName),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const callContext: CallContext = await evmUtil.getCallContext(
        bytecodeToBuffer(extcodesizeReplacement)
      )

      // Should pass calldata in the form that the execution manager expects:
      //  *       [methodID (bytes4)]
      //  *       [targetOvmContractAddress (address as bytes32 (big-endian))]
      //  *       [index (uint (32)]
      //  *       [length (uint (32))]
      const expectedCalldata: Buffer = Buffer.concat([
        abiForMethod.methodID(getMethodName, []),
        Buffer.alloc(12), // padding for 20-byte address
        hexStrToBuf(addressToRequest),
        bufferUtils.numberToBuffer(offset),
        bufferUtils.numberToBuffer(length),
      ])
      callContext.callData.equals(expectedCalldata).should.be.true

      // should call with the correct return memory values
      callContext.input.retOffset.should.equal(destOffset)
      callContext.input.retLength.should.equal(length)
    })
  })
})
