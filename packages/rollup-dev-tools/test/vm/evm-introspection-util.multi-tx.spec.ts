/* External Imports */
import { ethers } from 'ethers'
export const abi = new ethers.utils.AbiCoder()
import {
  add0x,
  bufToHexString,
  isValidHexAddress,
  remove0x,
} from '@pigi/core-utils'
import { Address } from '@pigi/rollup-core'

/* Internal Imports */
import { should } from '../setup'
import { EvmIntrospectionUtil, ExecutionResult } from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import * as SimpleCallable from '../contracts/build/SimpleCallable.json'

describe('EvmIntrospectionUtil', () => {
  let evmUtil: EvmIntrospectionUtil
  let contractBytecode: Buffer
  const functionName: string = 'update'
  const contractDeployParams: Buffer = Buffer.from(
    remove0x(abi.encode(['bytes'], ['0xdeadbeef'])),
    'hex'
  )

  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
    contractBytecode = Buffer.from(SimpleCallable.bytecode, 'hex')
  })

  const deployContract = async (
    util: EvmIntrospectionUtil
  ): Promise<Address> => {
    const result: ExecutionResult = await util.deployContract(
      contractBytecode,
      contractDeployParams
    )

    should.exist(result, 'Result should always exist!')
    should.not.exist(result.error, 'Result error mismatch!')
    should.exist(result.result, 'Result result mismatch!')

    const address: Address = bufToHexString(result.result)
    isValidHexAddress(address).should.equal(true, 'Invalid address!')
    return address
  }

  describe('deployContract', () => {
    it('should deploy simple contract successfully', async () => {
      await deployContract(evmUtil)
    })
  })

  describe('callContract', () => {
    let address: Address

    beforeEach(async () => {
      address = await deployContract(evmUtil)
    })

    it('should call deployed simple contract successfully', async () => {
      const result: ExecutionResult = await evmUtil.callContract(
        address,
        functionName,
        ['bytes'],
        contractDeployParams
      )

      should.exist(result, 'Result should never be empty!')
      should.not.exist(result.error, 'Error mismatch!')
      result.result.should.eql(contractDeployParams, 'Result mismatch!')
    })
  })
})
