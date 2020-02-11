/* External Imports */
import { ethers } from 'ethers'
import {
  bufToHexString,
  remove0x,
  getLogger,
  hexStrToBuf,
} from '@eth-optimism/core-utils'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  EVMOpcode,
  formatBytecode,
  Opcode,
  EVMOpcodeAndBytes,
} from '@eth-optimism/rollup-core'
import * as ethereumjsAbi from 'ethereumjs-abi'
import * as SimpleJumper from '../contracts/build/SimpleJumper.json'

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
import {
  ErroredTranspilation,
  OpcodeReplacer,
  OpcodeWhitelist,
  TranspilationErrors,
  TranspilationResult,
  Transpiler,
  SuccessfulTranspilation,
} from '../../src/types/transpiler'
import {
  TranspilerImpl,
  OpcodeReplacerImpl,
  OpcodeWhitelistImpl,
} from '../../src/tools/transpiler'
import {
  invalidBytesConsumedBytecode,
  invalidOpcode,
  multipleErrors,
  multipleNonWhitelisted,
  singleNonWhitelisted,
  stateManagerAddress,
  validBytecode,
} from '../helpers'

const log = getLogger(`test-solidity-JUMPs`)
const abi = new ethers.utils.AbiCoder()

describe('JUMP table solidity integration', () => {
  let evmUtil: EvmIntrospectionUtil
  const mockReplacer: OpcodeReplacer = {
    replaceIfNecessary(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
      if (opcodeAndBytes.opcode === Opcode.TIMESTAMP) {
        return [
          getPUSHIntegerOp(1),
          {
            opcode: Opcode.POP,
            consumedBytes: undefined,
          },
          getPUSHIntegerOp(2),
          {
            opcode: Opcode.POP,
            consumedBytes: undefined,
          },
          getPUSHIntegerOp(3),
          {
            opcode: Opcode.POP,
            consumedBytes: undefined,
          },
          {
            opcode: Opcode.TIMESTAMP,
            consumedBytes: undefined,
          },
        ]
      } else {
        return [opcodeAndBytes]
      }
    },
  }
  const opcodeWhitelist = new OpcodeWhitelistImpl(Opcode.ALL_OP_CODES)
  const transpiler = new TranspilerImpl(opcodeWhitelist, mockReplacer)

  const originalJumperAddr: Address =
    '0x1234123412341234123412341234123412341234'
  const transpiledJumperAddr: Address =
    '0x3456345634563456345634563456345634563456'
  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
    const originalJumperDeployedBytecde: Buffer = hexStrToBuf(
      SimpleJumper.evm.deployedBytecode.object
    )
    await evmUtil.deployBytecodeToAddress(
      originalJumperDeployedBytecde,
      hexStrToBuf(originalJumperAddr)
    )
    const transpiledJumperDeployedBytecode: Buffer = (transpiler.transpileRawBytecode(
      originalJumperDeployedBytecde
    ) as SuccessfulTranspilation).bytecode
    await evmUtil.deployBytecodeToAddress(
      transpiledJumperDeployedBytecode,
      hexStrToBuf(transpiledJumperAddr)
    )
  })
  it('should handle an if(true)', async () => {
    assertCallsProduceSameResult(
      evmUtil,
      originalJumperAddr,
      transpiledJumperAddr,
      'staticIfTrue'
    )
  })
  it('should handle an if(false)', async () => {
    assertCallsProduceSameResult(
      evmUtil,
      originalJumperAddr,
      transpiledJumperAddr,
      'staticIfFalseElse'
    )
  })
  it('should handle for loops', async () => {
    assertCallsProduceSameResult(
      evmUtil,
      originalJumperAddr,
      transpiledJumperAddr,
      'doForLoop'
    )
  })
  it('should handle while loops', async () => {
    assertCallsProduceSameResult(
      evmUtil,
      originalJumperAddr,
      transpiledJumperAddr,
      'doWhileLoop'
    )
  })
  it('should handle a while loop whose inner function calls another method with a for loop', async () => {
    assertCallsProduceSameResult(
      evmUtil,
      originalJumperAddr,
      transpiledJumperAddr,
      'doLoopingSubcalls'
    )
  })
  it('should handle a combination of a ton of these conditionals, subcalls, and loops', async () => {
    const nonzeroInput = '0x123456'
    const paramTypes = ['uint256']
    const callParams: Buffer = Buffer.from(
      remove0x(abi.encode(paramTypes, [nonzeroInput])),
      'hex'
    )
    assertCallsProduceSameResult(
      evmUtil,
      originalJumperAddr,
      transpiledJumperAddr,
      'doCrazyCombination',
      paramTypes,
      callParams
    )
  })
})

const assertCallsProduceSameResult = async (
  util: EvmIntrospectionUtil,
  addr1: Address,
  addr2: Address,
  methodName?: string,
  paramTypes?: string[],
  abiEncodedParams?: Buffer
) => {
  const res1 = await util.callContract(
    addr1,
    methodName,
    paramTypes,
    abiEncodedParams
  )
  if (res1.error) {
    throw new Error(
      `TEST ERROR: failed to execute callContract() for contract address: ${addr1}.  Error was: \n${res1.error}`
    )
  }
  const res2 = await util.callContract(
    addr2,
    methodName,
    paramTypes,
    abiEncodedParams
  )
  if (res2.error) {
    throw new Error(
      `TEST ERROR: failed to execute callContract() for contract address: ${addr2}.  Error was: \n${res2.error}`
    )
  }
  res2.result.should.deep.equal(res1.result)
}
