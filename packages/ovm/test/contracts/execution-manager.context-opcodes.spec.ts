/* External Imports */
import { Address } from '@pigi/rollup-core'
import {
  bufferUtils,
  bufToHexString,
  getLogger,
  remove0x,
  TestUtils,
} from '@pigi/core-utils'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as ContextContract from '../../build/contracts/ContextContract.json'
import * as PurityChecker from '../../build/contracts/PurityChecker.json'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  getUnsignedTransactionCalldata,
  bytes32AddressToAddress,
  addressTobytes32Address,
} from '../helpers'
import { should } from '../setup'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-context', true)

const executeCallMethodId: string = ethereumjsAbi
  .methodID('executeCall', [])
  .toString('hex')

const callThroughEMMethodId: string = ethereumjsAbi
  .methodID('callThroughExecutionManager', [])
  .toString('hex')

/*********
 * TESTS *
 *********/

describe('Execution Manager -- Context opcodes', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  const defaultTimestampAndQueueOrigin: string = '00'.repeat(64)
  const ONE_FILLED_BYTES_32 = '0x' + '11'.repeat(32)

  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let purityChecker: Contract
  let contract: ContractFactory
  let contract2: ContractFactory
  let contractAddress: Address
  let contract2Address: Address

  let contractAddress32: string
  let contract2Address32: string

  /* Link libraries before tests */
  before(async () => {
    purityChecker = await deployContract(
      wallet,
      PurityChecker,
      [ONE_FILLED_BYTES_32],
      { gasLimit: 6700000 }
    )
  })

  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and DummyContract

    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [purityChecker.address, '0x' + '00'.repeat(20)],
      { gasLimit: 6700000 }
    )

    // Deploy SimpleCopier with the ExecutionManager
    contractAddress = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      ContextContract,
      [executionManager.address]
    )

    log.debug(`Contract address: [${contractAddress}]`)

    // Also set our simple copier Ethers contract so we can generate unsigned transactions
    contract = new ContractFactory(
      ContextContract.abi as any,
      ContextContract.bytecode
    )

    // Deploy SimpleCopier with the ExecutionManager
    contract2Address = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      ContextContract,
      [executionManager.address]
    )

    log.debug(`Contract address: [${contractAddress}]`)

    // Also set our simple copier Ethers contract so we can generate unsigned transactions
    contract2 = new ContractFactory(
      ContextContract.abi as any,
      ContextContract.bytecode
    )

    contractAddress32 = addressTobytes32Address(contractAddress)
    contract2Address32 = addressTobytes32Address(contract2Address)
  })

  describe('ovmCALLER', async () => {
    it('reverts when CALLER is not set', async () => {
      const callerMethodId: string = ethereumjsAbi
        .methodID('ovmCALLER', [])
        .toString('hex')

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contractAddress32
      )}${callerMethodId}`

      await TestUtils.assertThrowsAsync(async () => {
        await executionManager.provider.call({
          to: executionManager.address,
          data,
          gasLimit: 6_700_000,
        })
      })
    })

    it('properly retrieves CALLER when caller is set', async () => {
      const callerMethodId: string = ethereumjsAbi
        .methodID('getCALLER', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contract2Address32
      )}${callerMethodId}`

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contractAddress32
      )}${internalCall}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`CALLER result: ${result}`)

      should.exist(result, 'Result should exist!')
      result.should.equal(contractAddress32, 'Addresses do not match.')
    })
  })

  describe('ovmADDRESS', async () => {
    it('reverts when ADDRESS is not set', async () => {
      const addressMethodId: string = ethereumjsAbi
        .methodID('ovmADDRESS', [])
        .toString('hex')

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contractAddress32
      )}${addressMethodId}`

      await TestUtils.assertThrowsAsync(async () => {
        await executionManager.provider.call({
          to: executionManager.address,
          data,
          gasLimit: 6_700_000,
        })
      })
    })

    it('properly retrieves ADDRESS when address is set', async () => {
      const addressMethodId: string = ethereumjsAbi
        .methodID('getADDRESS', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contract2Address32
      )}${addressMethodId}`

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contractAddress32
      )}${internalCall}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`ADDRESS result: ${result}`)

      should.exist(result, 'Result should exist!')
      result.should.equal(contract2Address32, 'Addresses do not match.')
    })
  })

  describe('ovmTIMESTAMP', async () => {
    it('reverts when TIMESTAMP is not set', async () => {
      const timestampMethodId: string = ethereumjsAbi
        .methodID('getTIMESTAMP', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contract2Address32
      )}${timestampMethodId}`

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contractAddress32
      )}${internalCall}`

      await TestUtils.assertThrowsAsync(async () => {
        await executionManager.provider.call({
          to: executionManager.address,
          data,
          gasLimit: 6_700_000,
        })
      })
    })

    it('properly retrieves TIMESTAMP when timestamp is set', async () => {
      const timestampMethodId: string = ethereumjsAbi
        .methodID('getTIMESTAMP', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contractAddress32
      )}${timestampMethodId}`

      const timestamp: string = '00'.repeat(30) + '1111'
      const queueOrigin: string = '00'.repeat(32)

      const data = `0x${executeCallMethodId}${timestamp}${queueOrigin}${remove0x(
        contract2Address32
      )}${internalCall}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`TIMESTAMP result: ${result}`)

      should.exist(result, 'Result should exist!')
      remove0x(result).should.equal(timestamp, 'Timestamps do not match.')
    })
  })

  describe('ovmGASLIMIT', async () => {
    it('properly retrieves GASLIMIT', async () => {
      const gasLimitMethodId: string = ethereumjsAbi
        .methodID('getGASLIMIT', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contractAddress32
      )}${gasLimitMethodId}`

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contract2Address32
      )}${internalCall}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`GASLIMIT result: ${result}`)

      should.exist(result, 'Result should exist!')
      const expected: string = bufToHexString(
        bufferUtils.numberToBuffer(100_000_000)
      )
      result.should.equal(expected, 'Gas limits do not match.')
    })
  })

  describe('ovmFraudProofGasLimit', async () => {
    it('properly retrieves fraudProofGasLimit', async () => {
      const gasLimitMethodId: string = ethereumjsAbi
        .methodID('getFraudProofGasLimit', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contractAddress32
      )}${gasLimitMethodId}`

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contract2Address32
      )}${internalCall}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`Fraud Proof Gas Limit result: ${result}`)

      should.exist(result, 'Result should exist!')
      const expected: string = bufToHexString(
        bufferUtils.numberToBuffer(50_000_000)
      )
      result.should.equal(expected, 'Gas limits do not match.')
    })
  })

  describe('ovmQueueOrigin', async () => {
    it('gets Queue Origin when it is 0', async () => {
      const timestampMethodId: string = ethereumjsAbi
        .methodID('getQueueOrigin', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contract2Address32
      )}${timestampMethodId}`

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contractAddress32
      )}${internalCall}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`QUEUE ORIGIN result: ${result}`)

      should.exist(result, 'Result should exist!')
      remove0x(result).should.equal(
        defaultTimestampAndQueueOrigin.substr(64),
        'Queue origins do not match.'
      )
    })

    it('properly retrieves Queue Origin when queue origin is set', async () => {
      const timestampMethodId: string = ethereumjsAbi
        .methodID('getQueueOrigin', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contractAddress32
      )}${timestampMethodId}`

      const timestamp: string = '00'.repeat(32)
      const queueOrigin: string = '00'.repeat(30) + '1111'

      const data = `0x${executeCallMethodId}${timestamp}${queueOrigin}${remove0x(
        contract2Address32
      )}${internalCall}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`QUEUE ORIGIN result: ${result}`)

      should.exist(result, 'Result should exist!')
      remove0x(result).should.equal(queueOrigin, 'Queue origins do not match.')
    })
  })
})
