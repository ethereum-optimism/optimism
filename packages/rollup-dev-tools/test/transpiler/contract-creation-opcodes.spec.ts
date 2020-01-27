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
import * as abiForMethod from 'ethereumjs-abi'

/* Internal Imports */
import {
  EvmIntrospectionUtil,
  ExecutionResult,
  StepContext,
  CallContext,
} from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import { setMemory, setupStackAndCALL } from '../helpers'
import { getCREATEReplacement, getCREATE2Replacement } from '../../src'

const log = getLogger(`test-static-memory-opcodes`)

const abi = new ethers.utils.AbiCoder()

/* Contracts */
import * as AssemblyReturnGetter from '../contracts/build/AssemblyReturnGetter.json'
import {
  getPUSHBuffer,
  getPUSHIntegerOp,
} from '../../src/tools/transpiler/memory-substitution'

const valToReturn =
  '0xbeadfeedbeadfeedbeadfeedbeadfeedbeadfeedbeadfeedbeadfeedbeadfeed'
const contractDeployParams: Buffer = Buffer.from(
  remove0x(abi.encode(['bytes'], [valToReturn])),
  'hex'
)

describe('Contract Creation Opcode Replacements', () => {
  let evmUtil: EvmIntrospectionUtil
  const getMethodName: string = 'get'

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
  let mockCREATEReplacement: EVMBytecode
  let mockCREATE2Replacement: EVMBytecode
  const mockMemory: Buffer = Buffer.alloc(32 * 10).fill(28)
  const initcodeOffset: number = 1 + 32 * 2 // must exceed 32 * 2 to do CREATE2 word prepending (methodId and salt)
  const initcodeLength: number = 2
  const salt: Buffer = hexStrToBuf(
    '0xaaaaacbdefacbdefaaaaacbdefacbdefaaaaacbdefacbdefaaaaacbdefacbdef'
  )
  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
    getterAddress = await deployGetterContract(evmUtil)
    // mock a transpiler-output replaced CREATE
    mockCREATEReplacement = [
      // fill memory with some random data so that we can confirm it was not modified
      ...setMemory(mockMemory),
      getPUSHIntegerOp(initcodeLength),
      getPUSHIntegerOp(initcodeOffset),
      getPUSHIntegerOp(0), // value input, will be ignored by transpiled bytecode
      ...getCREATEReplacement(getterAddress, getMethodName),
      { opcode: Opcode.RETURN, consumedBytes: undefined },
    ]
    // mock a transpiler-output replaced CREATE
    mockCREATE2Replacement = [
      // fill memory with some random data so that we can confirm it was not modified
      ...setMemory(mockMemory),
      getPUSHBuffer(salt),
      getPUSHIntegerOp(initcodeLength),
      getPUSHIntegerOp(initcodeOffset),
      getPUSHIntegerOp(0), // value input, will be ignored by transpiled bytecode
      ...getCREATE2Replacement(getterAddress, getMethodName),
      { opcode: Opcode.RETURN, consumedBytes: undefined },
    ]
  })

  describe('CREATE replacement', () => {
    it('should pass the right calldata', async () => {
      const callContext: CallContext = await evmUtil.getCallContext(
        bytecodeToBuffer(mockCREATEReplacement)
      )
      // check we generated the correct calldata
      const expectedCallData: Buffer = Buffer.concat([
        abiForMethod.methodID(getMethodName, []), // prepended methodId
        mockMemory.slice(initcodeOffset, initcodeOffset + initcodeLength), // original initcode data
      ])

      callContext.callData.equals(expectedCallData).should.be.true
    })
    it('Should end up with the right stack and memory after the CALL', async () => {
      // make sure the end state of memory is unaffected
      const finalContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(mockCREATEReplacement),
        bytecodeToBuffer(mockCREATEReplacement).length - 1
      )
      finalContext.memory.equals(mockMemory).should.be.true

      // check that returned address is only thing left on the stack
      finalContext.stackDepth.should.equal(1)
      finalContext.stack[0].should.deep.equal(hexStrToBuf(valToReturn))
    })
  })

  describe('CREATE2 replacement', () => {
    it('should pass the right calldata', async () => {
      const callContext: CallContext = await evmUtil.getCallContext(
        bytecodeToBuffer(mockCREATE2Replacement)
      )
      // check we generated the correct calldata
      const expectedCallData: Buffer = Buffer.concat([
        abiForMethod.methodID(getMethodName, []), // prepended methodId
        bufferUtils.padLeft(salt, 32), // prepended salt
        mockMemory.slice(initcodeOffset, initcodeOffset + initcodeLength), // original initcode data
      ])

      callContext.callData.equals(expectedCallData).should.be.true
    })
    it('Should end up with the right stack and memory after the CALL', async () => {
      // make sure the end state of memory is unaffected
      const finalContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(mockCREATE2Replacement),
        bytecodeToBuffer(mockCREATE2Replacement).length - 1
      )
      finalContext.memory.equals(mockMemory).should.be.true

      // check that returned address is only thing left on the stack
      finalContext.stackDepth.should.equal(1)
      finalContext.stack[0].should.deep.equal(hexStrToBuf(valToReturn))
    })
  })
})
