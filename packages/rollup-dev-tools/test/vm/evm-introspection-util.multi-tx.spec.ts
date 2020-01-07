/* External Imports */
import { ethers } from 'ethers'
import { bufToHexString, isValidHexAddress, remove0x } from '@pigi/core-utils'
import { Address, bytecodeToBuffer, EVMBytecode } from '@pigi/rollup-core'

/* Internal Imports */
import { should } from '../setup'
import {
  EvmErrors,
  EvmIntrospectionUtil,
  ExecutionResult,
} from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import {
  emptyBuffer,
  getBytecodeCallingContractMethod,
  invalidBytesConsumedBytecode,
} from '../helpers'

const abi = new ethers.utils.AbiCoder()

/* Contracts */
import * as SimpleCallable from '../contracts/build/SimpleCallable.json'

describe('EvmIntrospectionUtil', () => {
  let evmUtil: EvmIntrospectionUtil
  let contractBytecode: Buffer
  const updateFunctionName: string = 'update'
  const getterFunctionName: string = 'get'
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

    it('should fail on invalid bytecode', async () => {
      const result: ExecutionResult = await evmUtil.deployContract(
        bytecodeToBuffer(invalidBytesConsumedBytecode)
      )

      result.result.should.eql(emptyBuffer, 'Result should be empty!')
      result.error.should.eql(
        EvmErrors.STACK_UNDERFLOW_ERROR,
        'Invalid deploy should revert!'
      )
    })
  })

  describe('callContract', () => {
    let address: Address

    beforeEach(async () => {
      address = await deployContract(evmUtil)
    })

    it('should call deployed simple contract successfully -- without parameters', async () => {
      const result: ExecutionResult = await evmUtil.callContract(
        address,
        getterFunctionName
      )

      should.exist(result, 'Result should never be empty!')
      should.not.exist(result.error, 'Error mismatch!')
      result.result.should.eql(contractDeployParams, 'Result mismatch!')
    })

    it('should call deployed simple contract successfully -- With parameters', async () => {
      const result: ExecutionResult = await evmUtil.callContract(
        address,
        updateFunctionName,
        ['bytes'],
        contractDeployParams
      )

      should.exist(result, 'Result should never be empty!')
      should.not.exist(result.error, 'Error mismatch!')
      result.result.should.eql(contractDeployParams, 'Result mismatch!')
    })

    it('should fail to call invalid contract method', async () => {
      const result: ExecutionResult = await evmUtil.callContract(
        address,
        'derp'
      )

      should.exist(result, 'Result should never be empty!')
      should.exist(result.result, 'Result should always exist!')
      result.result.should.eql(emptyBuffer, 'Result mismatch!')
      should.exist(result.error, 'Error mismatch!')
      result.error.should.eql(EvmErrors.REVERT_ERROR, 'Result mismatch!')
    })

    it('should fail to call contract method with invalid param type', async () => {
      const result: ExecutionResult = await evmUtil.callContract(
        address,
        updateFunctionName,
        ['bytes32'],
        contractDeployParams
      )

      should.exist(result, 'Result should never be empty!')
      should.exist(result.result, 'Result should always exist!')
      result.result.should.eql(emptyBuffer, 'Result mismatch!')
      should.exist(result.error, 'Error mismatch!')
      result.error.should.eql(EvmErrors.REVERT_ERROR, 'Result mismatch!')
    })

    it('should fail to call contract method with invalid parameter', async () => {
      const result: ExecutionResult = await evmUtil.callContract(
        address,
        updateFunctionName,
        ['bytes'],
        emptyBuffer
      )

      should.exist(result, 'Result should never be empty!')
      should.exist(result.result, 'Result should always exist!')
      result.result.should.eql(emptyBuffer, 'Result mismatch!')
      should.exist(result.error, 'Error mismatch!')
      result.error.should.eql(EvmErrors.REVERT_ERROR, 'Result mismatch!')
    })

    it('should fail to call contract method expecting params without params', async () => {
      const result: ExecutionResult = await evmUtil.callContract(
        address,
        updateFunctionName
      )

      should.exist(result, 'Result should never be empty!')
      should.exist(result.result, 'Result should always exist!')
      result.result.should.eql(emptyBuffer, 'Result mismatch!')
      should.exist(result.error, 'Error mismatch!')
      result.error.should.eql(EvmErrors.REVERT_ERROR, 'Result mismatch!')
    })
  })

  describe('Deploy + Execute bytecode calling deployed contract', () => {
    let address: Address

    beforeEach(async () => {
      address = await deployContract(evmUtil)
    })

    it('should call deployed contract and return value', async () => {
      const bytecode: EVMBytecode = getBytecodeCallingContractMethod(
        address,
        getterFunctionName,
        contractDeployParams.length
      )
      const res: ExecutionResult = await evmUtil.getExecutionResult(
        bytecodeToBuffer(bytecode)
      )

      should.exist(res, 'Result should always exist!')
      should.not.exist(res.error, 'Error mismatch!')
      should.exist(res.result, 'Result should exist!')
      res.result.should.eql(contractDeployParams, 'Result mismatch!')
    })
  })
})
