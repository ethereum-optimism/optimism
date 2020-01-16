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

import { getCallTypeReplacement } from '../../src'

const log = getLogger(`test-static-memory-opcodes`)

import * as abiForMethod from 'ethereumjs-abi'
const abi = new ethers.utils.AbiCoder()

/* Contracts */
import * as AssemblyReturnGetter from '../contracts/build/AssemblyReturnGetter.json'
import {
  getPUSHBuffer,
  getPUSHIntegerOp,
} from '../../src/tools/transpiler/memory-substitution'

// Helper function, sets the memory to the given buffer
// TODO Move this into helpers and de-duplicate the one in static-mem-opcodes test
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

const contractDeployParams: Buffer = Buffer.from(
  remove0x(abi.encode(['bytes'], ['0xbeadfeed'])),
  'hex'
)

describe('Memory-dynamic Opcode Replacement', () => {
  let evmUtil: EvmIntrospectionUtil
  const callProxyFunctionName: string = 'get'

  const setupStackForCALL: EVMBytecode = [
    // fill memory with some random data so that we can confirm it was not modified
    ...setMemory(Buffer.alloc(32 * 10).fill(25)),
    getPUSHIntegerOp(5), // ret length
    getPUSHIntegerOp(8 * 32), // ret offset; must exceed 4 * 32, TODO: need to write new memory in a loop to fix this edge case?
    getPUSHIntegerOp(15), // args length
    getPUSHIntegerOp(4 * 32 + 17), // args offset; must exceed 4 * 32, TODO: need to write new memory in a loop to fix this edge case?
    getPUSHIntegerOp(0), // value
    getPUSHBuffer(hexStrToBuf('0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef')), // target address
    getPUSHIntegerOp(100100100), // gas
  ]

  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
  })

  const deployCallProxyContract = async (
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
  describe.only('Call-type opcode replacements', () => {
    it.only('should parse a CALL replacement', async () => {
      const getterAddress: Address = await deployCallProxyContract(evmUtil)
      const callReplacement: EVMBytecode = [
        ...setupStackForCALL,
        ...getCallTypeReplacement(getterAddress, callProxyFunctionName, true),
        { opcode: Opcode.RETURN, consumedBytes: undefined },
      ]
      const proxiedCall = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(callReplacement),
        566 // hardcoded  PC val, found via debug log
      )
    })
  })
})
