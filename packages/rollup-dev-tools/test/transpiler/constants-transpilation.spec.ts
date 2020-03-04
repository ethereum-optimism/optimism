/* External Imports */
import { ethers } from 'ethers'
import {
  bufToHexString,
  remove0x,
  getLogger,
  hexStrToBuf,
  deploy,
} from '@eth-optimism/core-utils'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  EVMOpcode,
  formatBytecode,
  Opcode,
  EVMOpcodeAndBytes,
  bufferToBytecode,
} from '@eth-optimism/rollup-core'
import * as ethereumjsAbi from 'ethereumjs-abi'
import * as ConstantGetter from '../contracts/build/ConstantGetter.json'

/* Internal Imports */
import {
  EvmIntrospectionUtil,
  ExecutionResult,
  StepContext,
  CallContext,
  EvmIntrospectionUtilImpl,
  getPUSHBuffer,
  getPUSHIntegerOp,
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
import { transpileAndDeployInitcode } from '../helpers'

const log = getLogger(`test-constructor-params`)
const abi = new ethers.utils.AbiCoder()

const getGetterReturnedVal = async (
  deployedAddress: Buffer,
  methodId: string,
  evmUtil: EvmIntrospectionUtil
): Promise<Buffer> => {
  const callRes: ExecutionResult = await evmUtil.callContract(
    bufToHexString(deployedAddress),
    methodId
  )
  if (!!callRes.error) {
    throw new Error(
      `call to retrieveStoredVal() failed with evmUtil Error: ${callRes.error}`
    )
  }
  return callRes.result
}

describe('Solitity contracts should have constants correctly accessible when using transpiled initcode', () => {
  let evmUtil: EvmIntrospectionUtil
  const mockReplacer: OpcodeReplacer = {
    replaceIfNecessary(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
      if (opcodeAndBytes.opcode === Opcode.SSTORE) {
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
            opcode: Opcode.SSTORE,
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
  let deployedGetterAddress: Buffer
  const randomUncheckedParam = ['0x1234123412341234']
  const randomUncheckedParamEncoding = ['bytes']
  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
    deployedGetterAddress = await transpileAndDeployInitcode(
      ConstantGetter,
      randomUncheckedParam,
      randomUncheckedParamEncoding,
      transpiler,
      evmUtil
    )
  })

  const bytes32Const: Buffer = hexStrToBuf(
    '0xABCDEF34ABCDEF34ABCDEF34ABCDEF34ABCDEF34ABCDEF34ABCDEF34ABCDEF34'
  )
  it('should work for a bytes32 constant', async () => {
    const code: Buffer = await evmUtil.getContractDeployedBytecode(
      deployedGetterAddress
    )
    log.debug(`deployed code is: \n${formatBytecode(bufferToBytecode(code))}`)
    const retrievedBytes32Val: Buffer = await getGetterReturnedVal(
      deployedGetterAddress,
      'getBytes32Constant',
      evmUtil
    )
    retrievedBytes32Val.should.deep.equal(bytes32Const)
  })

  const bytesMemoryConstA: Buffer = hexStrToBuf(
    '0xAAAdeadbeefAAAAAAdeadbeefAAAAAAdeadbeefAAAAAAdeadbeefAAAAAAdeadbeefAAAAAAdeadbeefAAAAAAdeadbeefAAA'
  )
  it('should work for the first bytes memory constant', async () => {
    const retrievedBytesMemoryAVal: Buffer = await getGetterReturnedVal(
      deployedGetterAddress,
      'getBytesMemoryConstantA',
      evmUtil
    )
    const encodedBytesMemoryConstA: Buffer = hexStrToBuf(
      abi.encode(['bytes'], [bufToHexString(bytesMemoryConstA)])
    )
    retrievedBytesMemoryAVal.should.deep.equal(encodedBytesMemoryConstA)
  })

  const bytesMemoryConstB: Buffer = Buffer.from(
    `this should pass but the error message is much longer`
  )
  it('should work for the second bytes memory constant', async () => {
    const retrievedBytesMemoryBVal: Buffer = await getGetterReturnedVal(
      deployedGetterAddress,
      'getBytesMemoryConstantB',
      evmUtil
    )
    const encodedbytesMemoryConstB: Buffer = hexStrToBuf(
      abi.encode(['bytes'], [bufToHexString(bytesMemoryConstB)])
    )
    retrievedBytesMemoryBVal.should.deep.equal(encodedbytesMemoryConstB)
  })
})
