import '../setup'

/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import {
  getLogger,
  BigNumber,
  remove0x,
  add0x,
  TestUtils,
} from '@eth-optimism/core-utils'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as DummyContract from '../../build/contracts/DummyContract.json'
import * as SimpleCall from '../../build/contracts/SimpleCall.json'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  addressToBytes32Address,
  DEFAULT_ETHNODE_GAS_LIMIT,
  didCreateSucceed,
  gasLimit,
} from '../helpers'
import { GAS_LIMIT, OPCODE_WHITELIST_MASK } from '../../src/app'
import { TransactionReceipt } from 'ethers/providers'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-calls', true)

/*********
 * TESTS *
 *********/

const executeCallMethodId: string = ethereumjsAbi
  .methodID('executeCall', [])
  .toString('hex')

const sstoreMethodId: string = ethereumjsAbi
  .methodID('notStaticFriendlySSTORE', [])
  .toString('hex')

const createMethodId: string = ethereumjsAbi
  .methodID('notStaticFriendlyCREATE', [])
  .toString('hex')

const create2MethodId: string = ethereumjsAbi
  .methodID('notStaticFriendlyCREATE2', [])
  .toString('hex')

const sloadMethodId: string = ethereumjsAbi
  .methodID('staticFriendlySLOAD', [])
  .toString('hex')

const staticCallThenCallMethodId: string = ethereumjsAbi
  .methodID('makeStaticCallThenCall', [])
  .toString('hex')

const sloadKey: string = '11'.repeat(32)
const unpopultedSLOADResult: string = '00'.repeat(32)
const populatedSLOADResult: string = '22'.repeat(32)

const sstoreMethodIdAndParams: string = `${sstoreMethodId}${sloadKey}${populatedSLOADResult}`
const sloadMethodIdAndParams: string = `${sloadMethodId}${sloadKey}`

const timestampAndQueueOrigin: string = '00'.repeat(64)

describe('Execution Manager -- Call opcodes', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let dummyContract: Contract
  let callContract: ContractFactory
  let callContractAddress: Address
  let callContract2Address: Address
  let callContract3Address: Address
  let callContractAddress32: string
  let callContract2Address32: string
  let callContract3Address32: string
  let executeCallToCallContractData: string

  let createMethodIdAndData: string
  let create2MethodIdAndData: string

  /* Link libraries before tests */
  before(async () => {
    dummyContract = await deployContract(wallet, DummyContract, [], {
      gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
    })

    const deployTx: any = new ContractFactory(
      SimpleCall.abi,
      SimpleCall.bytecode
    ).getDeployTransaction(dummyContract.address)

    createMethodIdAndData = `${createMethodId}${remove0x(deployTx.data)}`
    create2MethodIdAndData = `${create2MethodId}${'00'.repeat(32)}${remove0x(
      deployTx.data
    )}`
  })
  beforeEach(async () => {
    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      {
        gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
      }
    )

    // Deploy SimpleCall with the ExecutionManager
    callContractAddress = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      SimpleCall,
      [executionManager.address]
    )

    // Deploy second SimpleCall contract
    callContract2Address = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      SimpleCall,
      [executionManager.address]
    )

    // Deploy third SimpleCall contract
    callContract3Address = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      SimpleCall,
      [executionManager.address]
    )

    log.debug(`Contract address: [${callContractAddress}]`)

    // Also set our simple copier Ethers contract so we can generate unsigned transactions
    callContract = new ContractFactory(
      SimpleCall.abi as any,
      SimpleCall.bytecode
    )

    callContractAddress32 = remove0x(
      addressToBytes32Address(callContractAddress)
    )
    callContract2Address32 = remove0x(
      addressToBytes32Address(callContract2Address)
    )
    callContract3Address32 = remove0x(
      addressToBytes32Address(callContract2Address)
    )
    const encodedParams = `${timestampAndQueueOrigin}${callContractAddress32}`
    executeCallToCallContractData = `0x${executeCallMethodId}${encodedParams}`
  })

  describe('ovmCALL', async () => {
    const callMethodId: string = ethereumjsAbi
      .methodID('makeCall', [])
      .toString('hex')

    it('properly executes ovmCALL to SLOAD', async () => {
      const data: string = `${executeCallToCallContractData}${callMethodId}${callContract2Address32}${sloadMethodIdAndParams}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })

      log.debug(`Result: [${result}]`)

      remove0x(result).should.equal(unpopultedSLOADResult, 'Result mismatch!')
    })

    it('properly executes ovmCALL to SSTORE', async () => {
      const data: string = `${executeCallToCallContractData}${callMethodId}${callContract2Address32}${sstoreMethodIdAndParams}`

      // Note: Send transaction vs call so it is persisted
      await wallet.sendTransaction({
        to: executionManager.address,
        data,
        gasLimit,
      })

      const fetchData: string = `${executeCallToCallContractData}${callMethodId}${callContract2Address32}${sloadMethodIdAndParams}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data: fetchData,
        gasLimit,
      })

      log.debug(`Result: [${result}]`)

      // Stored in contract 2, matches contract 2
      remove0x(result).should.equal(populatedSLOADResult, 'SLOAD mismatch!')
    })

    it('properly executes ovmCALL to CREATE', async () => {
      const data: string = `${executeCallToCallContractData}${callMethodId}${callContract2Address32}${createMethodIdAndData}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })

      log.debug(`RESULT: ${result}`)

      const address = remove0x(result)
      address.length.should.equal(64, 'Should have got a bytes32 address back!')
      address.length.should.not.equal(
        '00'.repeat(32),
        'Should not be 0 address!'
      )
    })

    it('properly executes ovmCALL to CREATE2', async () => {
      const data: string = `${executeCallToCallContractData}${callMethodId}${callContract2Address32}${create2MethodIdAndData}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })

      log.debug(`RESULT: ${result}`)

      const address = remove0x(result)
      address.length.should.equal(64, 'Should have got a bytes32 address back!')
      address.length.should.not.equal(
        '00'.repeat(32),
        'Should not be 0 address!'
      )
    })
  })

  describe('ovmDELEGATECALL', async () => {
    const delegateCallMethodId: string = ethereumjsAbi
      .methodID('makeDelegateCall', [])
      .toString('hex')

    const callMethodId: string = ethereumjsAbi
      .methodID('makeCall', [])
      .toString('hex')

    it('properly executes ovmDELEGATECALL to SSTORE', async () => {
      const data: string = `${executeCallToCallContractData}${delegateCallMethodId}${callContract2Address32}${sstoreMethodIdAndParams}`

      // Note: Send transaction vs call so it is persisted
      await wallet.sendTransaction({
        to: executionManager.address,
        data,
        gasLimit,
      })

      // Stored in contract 2 via delegate call but accessed via contract 1
      const fetchData: string = `${executeCallToCallContractData}${callMethodId}${callContractAddress32}${sloadMethodIdAndParams}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data: fetchData,
        gasLimit,
      })

      log.debug(`Result: [${result}]`)
      // Should have stored result
      remove0x(result).should.equal(
        populatedSLOADResult,
        'SLOAD should yield stored result!'
      )

      const contract2FetchData: string = `${executeCallToCallContractData}${callMethodId}${callContract2Address32}${sloadMethodIdAndParams}`
      const contract2Result = await executionManager.provider.call({
        to: executionManager.address,
        data: contract2FetchData,
        gasLimit,
      })

      log.debug(`Result: [${contract2Result}]`)

      // Should not be stored
      remove0x(contract2Result).should.equal(
        unpopultedSLOADResult,
        'SLOAD should not yield any data (0 x 32 bytes)!'
      )
    })

    it('properly executes nested ovmDELEGATECALLs to SSTORE', async () => {
      // contract 1 delegate calls contract 2 delegate calls contract 3
      const data: string = `${executeCallToCallContractData}${delegateCallMethodId}${callContract2Address32}${delegateCallMethodId}${callContract3Address32}${sstoreMethodIdAndParams}`

      // Note: Send transaction vs call so it is persisted
      await wallet.sendTransaction({
        to: executionManager.address,
        data,
        gasLimit,
      })

      const contract1FetchData: string = `${executeCallToCallContractData}${callMethodId}${callContractAddress32}${sloadMethodIdAndParams}`
      const contract1Result = await executionManager.provider.call({
        to: executionManager.address,
        data: contract1FetchData,
        gasLimit,
      })

      log.debug(`Result 1: [${contract1Result}]`)

      // Stored in contract 3 via delegate call but accessed via contract 1
      remove0x(contract1Result).should.equal(
        populatedSLOADResult,
        'SLOAD should yield stored data!'
      )

      const contract2FetchData: string = `${executeCallToCallContractData}${callMethodId}${callContract2Address32}${sloadMethodIdAndParams}`
      const contract2Result = await executionManager.provider.call({
        to: executionManager.address,
        data: contract2FetchData,
        gasLimit,
      })

      log.debug(`Result 2: [${contract2Result}]`)

      // Should not be stored
      remove0x(contract2Result).should.equal(
        unpopultedSLOADResult,
        'SLOAD should not yield any data (0 x 32 bytes)!'
      )

      const contract3FetchData: string = `${executeCallToCallContractData}${callMethodId}${callContract3Address32}${sloadMethodIdAndParams}`
      const contract3Result = await executionManager.provider.call({
        to: executionManager.address,
        data: contract3FetchData,
        gasLimit,
      })

      log.debug(`Result 3: [${contract3Result}]`)

      // Should not be stored
      remove0x(contract3Result).should.equal(
        unpopultedSLOADResult,
        'SLOAD should not yield any data (0 x 32 bytes)!'
      )
    })
  })

  describe('ovmSTATICCALL', async () => {
    const staticCallMethodId: string = ethereumjsAbi
      .methodID('makeStaticCall', [])
      .toString('hex')

    it('properly executes ovmSTATICCALL to SLOAD', async () => {
      const data: string = `${executeCallToCallContractData}${staticCallMethodId}${callContract2Address32}${sloadMethodIdAndParams}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })

      log.debug(`Result: [${result}]`)

      remove0x(result).should.equal(unpopultedSLOADResult, 'Result mismatch!')
    })

    it('properly executes nested ovmSTATICCALL to SLOAD', async () => {
      const data: string = `${executeCallToCallContractData}${staticCallMethodId}${callContract2Address32}${staticCallMethodId}${callContract2Address32}${sloadMethodIdAndParams}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })

      log.debug(`Result: [${result}]`)

      remove0x(result).should.equal(unpopultedSLOADResult, 'Result mismatch!')
    })

    it('successfully makes static call then call', async () => {
      const data: string = `${executeCallToCallContractData}${staticCallThenCallMethodId}${callContractAddress32}`

      // Should not throw
      await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })
    })

    it('remains in static context when exiting nested static context', async () => {
      const data: string = `${executeCallToCallContractData}${staticCallMethodId}${callContractAddress32}${staticCallThenCallMethodId}${callContractAddress32}`

      await TestUtils.assertThrowsAsync(async () => {
        const res = await executionManager.provider.call({
          to: executionManager.address,
          data,
          gasLimit,
        })
      })
    })

    it('fails on ovmSTATICCALL to SSTORE', async () => {
      const data: string = `${executeCallToCallContractData}${staticCallMethodId}${callContract2Address32}${sstoreMethodIdAndParams}`

      await TestUtils.assertThrowsAsync(async () => {
        // Note: Send transaction vs call so it is persisted
        await wallet.sendTransaction({
          to: executionManager.address,
          data,
          gasLimit,
        })
      })
    })

    it('Fails to create on ovmSTATICCALL to CREATE -- tx', async () => {
      const data: string = `${executeCallToCallContractData}${staticCallMethodId}${callContract2Address32}${createMethodIdAndData}`

      // Note: Send transaction vs call so it is persisted
      const receipt = await wallet.sendTransaction({
        to: executionManager.address,
        data,
        gasLimit,
      })

      const createSucceeded = await didCreateSucceed(
        executionManager,
        receipt.hash
      )
      createSucceeded.should.equal(false, 'Create should have failed!')
    })

    it('Fails to create on ovmSTATICCALL to CREATE -- call', async () => {
      const data: string = `${executeCallToCallContractData}${staticCallMethodId}${callContract2Address32}${createMethodIdAndData}`

      const res = await wallet.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })

      const address = remove0x(res)
      address.should.equal('00'.repeat(32), 'Should be 0 address!')
    })

    it('fails on ovmSTATICCALL to CREATE2 -- tx', async () => {
      const data: string = `${executeCallToCallContractData}${staticCallMethodId}${callContract2Address32}${create2MethodIdAndData}`

      // Note: Send transaction vs call so it is persisted
      const receipt = await wallet.sendTransaction({
        to: executionManager.address,
        data,
        gasLimit,
      })

      const createSucceeded = await didCreateSucceed(
        executionManager,
        receipt.hash
      )
      createSucceeded.should.equal(false, 'Create should have failed!')
    })

    it('fails on ovmSTATICCALL to CREATE2 -- call', async () => {
      const data: string = `${executeCallToCallContractData}${staticCallMethodId}${callContract2Address32}${create2MethodIdAndData}`

      const res = await wallet.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })

      const address = remove0x(res)
      address.should.equal('00'.repeat(32), 'Should be 0 address!')
    })
  })
})
