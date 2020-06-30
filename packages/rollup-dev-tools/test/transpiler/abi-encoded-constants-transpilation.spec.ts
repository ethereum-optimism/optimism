/* External Imports */
import { ethers } from 'ethers'
import { bufToHexString, getLogger } from '@eth-optimism/core-utils'
import {
  formatBytecode,
  Opcode,
  bufferToBytecode,
} from '@eth-optimism/rollup-core'
import * as AbiEncodedConstantInConstructor from '../contracts/build/AbiEncodedConstantInConstructor.json'

/* Internal Imports */
import {
  EvmIntrospectionUtil,
  ExecutionResult,
  EvmIntrospectionUtilImpl,
} from '../../src'

import { TranspilerImpl, OpcodeWhitelistImpl } from '../../src/tools/transpiler'
import { transpileAndDeployInitcode, mockSSTOREReplacer } from '../helpers'

const log = getLogger(`test-constructor-params-new`)

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

describe('Solitity contracts should have hardcoded values correctly accessible in transpiled initcode', () => {
  let evmUtil: EvmIntrospectionUtil

  const opcodeWhitelist = new OpcodeWhitelistImpl(Opcode.ALL_OP_CODES)
  const transpiler = new TranspilerImpl(opcodeWhitelist, mockSSTOREReplacer)
  let deployedGetterAddress: Buffer
  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
    log.debug(
      `transpiling and deploying initcode which should store hash in constructor`
    )
    deployedGetterAddress = await transpileAndDeployInitcode(
      AbiEncodedConstantInConstructor,
      [],
      [],
      transpiler,
      evmUtil
    )
    const code: Buffer = await evmUtil.getContractDeployedBytecode(
      deployedGetterAddress
    )
    log.debug(
      `Initcode transpiled and deployed.  The code is:\n${formatBytecode(
        bufferToBytecode(code)
      )}`
    )
  })

  it(`The hash of an abi.encode(hardcoded) should be correct and retrievable if stored during constructor()`, async () => {
    const expectedStoredVal = ethers.utils.keccak256(
      ethers.utils.defaultAbiCoder.encode(
        ['bytes32', 'uint256'],
        [
          ethers.utils.keccak256(
            Buffer.from(
              ethers.utils.toUtf8Bytes(
                'EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)'
              )
            )
          ),
          1,
        ]
      )
    )
    const res = await getGetterReturnedVal(
      deployedGetterAddress,
      'getConstant',
      evmUtil
    )
    bufToHexString(res).should.eq(expectedStoredVal)
  })
})
