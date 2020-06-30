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
import * as TestConstantsConstructor from '../contracts/build/TestConstantsConstructor.json'

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
import { transpileAndDeployInitcode, mockSSTOREReplacer } from '../helpers'

const log = getLogger(`test-constructor-params-new`)
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
      `call to ${methodId} failed with evmUtil Error: ${callRes.error}`
    )
  }
  return callRes.result
}

describe('Solitity contracts should have constants correctly accessible when using transpiled initcode', () => {
  let evmUtil: EvmIntrospectionUtil

  const opcodeWhitelist = new OpcodeWhitelistImpl(Opcode.ALL_OP_CODES)
  const transpiler = new TranspilerImpl(opcodeWhitelist, mockSSTOREReplacer)
  let deployedGetterAddress: Buffer
  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
    log.debug(`transpiling and deploying initcode which should store hash in constructor`)
    deployedGetterAddress = await transpileAndDeployInitcode(
      TestConstantsConstructor,
      [],
      [],
      transpiler,
      evmUtil
    )
    const code: Buffer = await evmUtil.getContractDeployedBytecode(
      deployedGetterAddress
    )
    log.debug(`Initcode transpiled and deployed.  The code is:\n${formatBytecode(bufferToBytecode(code))}`)
  })

  it(`The result of a transpiled set() and then get() should be correct`, async () => {
    // set up expected values step-by-step
    const expectedInnerBytesBeingHashed = Buffer.from(ethers.utils.toUtf8Bytes('EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)'))
    log.debug(`expectedInnerBytesBeingHashed: ${bufToHexString(expectedInnerBytesBeingHashed)}`)
    const expectedInnerHashRaw = ethers.utils.keccak256(expectedInnerBytesBeingHashed)
    log.debug(`expectedInnerHashRaw: ${expectedInnerHashRaw}`)
    const expectedInnerHashAndValEncoded = ethers.utils.defaultAbiCoder.encode(
      ['bytes32', 'uint256'],
      [
        expectedInnerHashRaw,
        1,
      ]
      )
    log.debug(`expectedInnerHashAndValEncoded: ${expectedInnerHashAndValEncoded}`)
    const expectedOuterHash = ethers.utils.keccak256(
      expectedInnerHashAndValEncoded
    )
    log.debug(`expected final outer hash: ${expectedOuterHash}`)
    log.debug(`Calling get...`)
    const res = await getGetterReturnedVal(
      deployedGetterAddress,
      'getConstant',
      evmUtil
    )
    bufToHexString(res).should.eq(expectedOuterHash)
  })
})
