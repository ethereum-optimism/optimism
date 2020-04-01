import '../setup'

/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import {
  getLogger,
  remove0x,
  add0x,
  TestUtils,
  getCurrentTime,
} from '@eth-optimism/core-utils'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

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
  executeOVMCall,
  encodeMethodId,
  encodeRawArguments,
} from '../helpers'
import { GAS_LIMIT, DEFAULT_OPCODE_WHITELIST_MASK } from '../../src/app'
import { fromPairs } from 'lodash'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-calls', true)

/*********
 * TESTS *
 *********/

const methodIds = fromPairs(
  [
    'executeCall',
    'makeCall',
    'makeStaticCall',
    'makeStaticCallThenCall',
    'staticFriendlySLOAD',
    'notStaticFriendlySSTORE',
    'notStaticFriendlyCREATE',
    'notStaticFriendlyCREATE2',
    'staticFriendlySLOAD',
    'makeDelegateCall',
  ].map((methodId) => [methodId, encodeMethodId(methodId)])
)

const sloadKey: string = '11'.repeat(32)
const unpopultedSLOADResult: string = '00'.repeat(32)
const populatedSLOADResult: string = '22'.repeat(32)

describe('Execution Manager -- Call opcodes', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let dummyContract: Contract
  let callContractAddress: Address
  let callContract2Address: Address
  let callContract3Address: Address
  let deployTx: any

  /* Link libraries before tests */
  before(async () => {
    dummyContract = await deployContract(wallet, DummyContract, [], {
      gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
    })

    deployTx = new ContractFactory(
      SimpleCall.abi,
      SimpleCall.bytecode
    ).getDeployTransaction(dummyContract.address)
  })
  beforeEach(async () => {
    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
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
  })

  describe('ovmCALL', async () => {
    it.only('properly executes ovmCALL to SLOAD', async () => {
      const result: string = await executeCall(
        callContractAddress,
        methodIds.staticFriendlySLOAD,
        [sloadKey]
      )
      log.debug(`Result: [${result}]`)

      remove0x(result).should.equal(unpopultedSLOADResult, 'Result mismatch!')
    })

    it('properly executes ovmCALL to SSTORE', async () => {
      const data: string =
        encodeMethodId('executeCall') +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeCall,
          addressToBytes32Address(callContract2Address),
          methodIds.notStaticFriendlySSTORE,
          sloadKey,
          populatedSLOADResult,
        ])

      // Note: Send transaction vs call so it is persisted
      await wallet.sendTransaction({
        to: executionManager.address,
        data: add0x(data),
        gasLimit,
      })

      const result: string = await executeCall(
        callContract2Address,
        methodIds.staticFriendlySLOAD,
        [sloadKey]
      )

      log.debug(`Result: [${result}]`)

      // Stored in contract 2, matches contract 2
      remove0x(result).should.equal(populatedSLOADResult, 'SLOAD mismatch!')
    })

    it('properly executes ovmCALL to CREATE', async () => {
      const result: string = await executeCall(
        callContract2Address,
        methodIds.notStaticFriendlyCREATE,
        [deployTx.data]
      )

      log.debug(`RESULT: ${result}`)

      const address = remove0x(result)
      address.length.should.equal(64, 'Should have got a bytes32 address back!')
      address.length.should.not.equal(
        '00'.repeat(32),
        'Should not be 0 address!'
      )
    })

    it('properly executes ovmCALL to CREATE2', async () => {
      const result: string = await executeCall(
        callContract2Address,
        methodIds.notStaticFriendlyCREATE2,
        [0, deployTx.data]
      )

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
    it('properly executes ovmDELEGATECALL to SSTORE', async () => {
      const data: string =
        methodIds.executeCall +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeDelegateCall,
          addressToBytes32Address(callContract2Address),
          methodIds.notStaticFriendlySSTORE,
          sloadKey,
          populatedSLOADResult,
        ])

      // Note: Send transaction vs call so it is persisted
      await wallet.sendTransaction({
        to: executionManager.address,
        data: add0x(data),
        gasLimit,
      })

      // Stored in contract 2 via delegate call but accessed via contract 1
      const result: string = await executeCall(
        callContractAddress,
        methodIds.staticFriendlySLOAD,
        [sloadKey]
      )

      log.debug(`Result: [${result}]`)
      // Should have stored result
      remove0x(result).should.equal(
        populatedSLOADResult,
        'SLOAD should yield stored result!'
      )

      const contract2Result: string = await executeCall(
        addressToBytes32Address(callContract2Address),
        methodIds.staticFriendlySLOAD,
        [sloadKey]
      )

      log.debug(`Result: [${contract2Result}]`)

      // Should not be stored
      remove0x(contract2Result).should.equal(
        unpopultedSLOADResult,
        'SLOAD should not yield any data (0 x 32 bytes)!'
      )
    })

    it('properly executes nested ovmDELEGATECALLs to SSTORE', async () => {
      // contract 1 delegate calls contract 2 delegate calls contract 3
      const data: string =
        methodIds.executeCall +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeDelegateCall,
          addressToBytes32Address(callContract2Address),
          methodIds.makeDelegateCall,
          addressToBytes32Address(callContract3Address),
          methodIds.notStaticFriendlySSTORE,
          sloadKey,
          populatedSLOADResult,
        ])

      // Note: Send transaction vs call so it is persisted
      await wallet.sendTransaction({
        to: executionManager.address,
        data: add0x(data),
        gasLimit,
      })

      const contract1Result: string = await executeCall(
        callContractAddress,
        methodIds.staticFriendlySLOAD,
        [sloadKey]
      )

      log.debug(`Result 1: [${contract1Result}]`)

      // Stored in contract 3 via delegate call but accessed via contract 1
      remove0x(contract1Result).should.equal(
        populatedSLOADResult,
        'SLOAD should yield stored data!'
      )

      const contract2Result: string = await executeCall(
        callContract2Address,
        methodIds.staticFriendlySLOAD,
        [sloadKey]
      )

      log.debug(`Result 2: [${contract2Result}]`)

      // Should not be stored
      remove0x(contract2Result).should.equal(
        unpopultedSLOADResult,
        'SLOAD should not yield any data (0 x 32 bytes)!'
      )

      const contract3Result: string = await executeCall(
        callContract3Address,
        methodIds.staticFriendlySLOAD,
        [sloadKey]
      )

      log.debug(`Result 3: [${contract3Result}]`)

      // Should not be stored
      remove0x(contract3Result).should.equal(
        unpopultedSLOADResult,
        'SLOAD should not yield any data (0 x 32 bytes)!'
      )
    })
  })

  describe('ovmSTATICCALL', async () => {
    it('properly executes ovmSTATICCALL to SLOAD', async () => {
      const result = await executeOVMCall(executionManager, 'executeCall', [
        0,
        0,
        addressToBytes32Address(callContractAddress),
        methodIds.makeStaticCall,
        addressToBytes32Address(callContract2Address),
        methodIds.staticFriendlySLOAD,
        sloadKey,
      ])

      log.debug(`Result: [${result}]`)

      remove0x(result).should.equal(unpopultedSLOADResult, 'Result mismatch!')
    })

    it('properly executes nested ovmSTATICCALL to SLOAD', async () => {
      const result = await executeOVMCall(executionManager, 'executeCall', [
        0,
        0,
        addressToBytes32Address(callContractAddress),
        methodIds.makeStaticCall,
        addressToBytes32Address(callContract2Address),
        methodIds.makeStaticCall,
        addressToBytes32Address(callContract2Address),
        methodIds.staticFriendlySLOAD,
        sloadKey,
      ])

      log.debug(`Result: [${result}]`)

      remove0x(result).should.equal(unpopultedSLOADResult, 'Result mismatch!')
    })

    it('successfully makes static call then call', async () => {
      const data: string =
        methodIds.executeCall +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeStaticCallThenCall,
          addressToBytes32Address(callContractAddress),
        ])

      // Should not throw
      await executionManager.provider.call({
        to: executionManager.address,
        data: add0x(data),
        gasLimit,
      })
    })

    it('remains in static context when exiting nested static context', async () => {
      const data: string =
        methodIds.executeCall +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeStaticCall,
          addressToBytes32Address(callContractAddress),
          methodIds.makeStaticCallThenCall,
          addressToBytes32Address(callContractAddress),
        ])

      await TestUtils.assertThrowsAsync(async () => {
        await executionManager.provider.call({
          to: executionManager.address,
          data,
          gasLimit,
        })
      })
    })

    it('fails on ovmSTATICCALL to SSTORE', async () => {
      const data: string =
        methodIds.executeCall +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeStaticCall,
          addressToBytes32Address(callContractAddress),
          methodIds.notStaticFriendlySSTORE,
          sloadKey,
          populatedSLOADResult,
        ])

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
      const data: string =
        methodIds.executeCall +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeStaticCall,
          addressToBytes32Address(callContractAddress),
          methodIds.notStaticFriendlyCREATE,
          deployTx.data,
        ])

      // Note: Send transaction vs call so it is persisted
      const receipt = await wallet.sendTransaction({
        to: executionManager.address,
        data: add0x(data),
        gasLimit,
      })

      const createSucceeded = await didCreateSucceed(
        executionManager,
        receipt.hash
      )
      createSucceeded.should.equal(false, 'Create should have failed!')
    })

    it('Fails to create on ovmSTATICCALL to CREATE -- call', async () => {
      const data: string =
        methodIds.executeCall +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeStaticCall,
          addressToBytes32Address(callContractAddress),
          methodIds.notStaticFriendlyCREATE,
          deployTx.data,
        ])

      const res = await wallet.provider.call({
        to: executionManager.address,
        data: add0x(data),
        gasLimit,
      })

      const address = remove0x(res)
      address.should.equal('00'.repeat(32), 'Should be 0 address!')
    })

    it('fails on ovmSTATICCALL to CREATE2 -- tx', async () => {
      const data: string =
        methodIds.executeCall +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeStaticCall,
          addressToBytes32Address(callContractAddress),
          methodIds.notStaticFriendlyCREATE2,
          0,
          deployTx.data,
        ])

      // Note: Send transaction vs call so it is persisted
      const receipt = await wallet.sendTransaction({
        to: executionManager.address,
        data: add0x(data),
        gasLimit,
      })

      const createSucceeded = await didCreateSucceed(
        executionManager,
        receipt.hash
      )
      createSucceeded.should.equal(false, 'Create should have failed!')
    })

    it('fails on ovmSTATICCALL to CREATE2 -- call', async () => {
      const data: string =
        methodIds.executeCall +
        encodeRawArguments([
          getCurrentTime(),
          0,
          addressToBytes32Address(callContractAddress),
          methodIds.makeStaticCall,
          addressToBytes32Address(callContractAddress),
          methodIds.notStaticFriendlyCREATE2,
          0,
          deployTx.data,
        ])

      const res = await wallet.provider.call({
        to: executionManager.address,
        data: add0x(data),
        gasLimit,
      })

      const address = remove0x(res)
      address.should.equal('00'.repeat(32), 'Should be 0 address!')
    })
  })

  const executeCall = async (
    contractAddress: string,
    methodId: string,
    args: any[]
  ): Promise<string> => {
    console.log(contractAddress)
    console.log(methodId)
    return executeOVMCall(executionManager, 'executeCall', [
      encodeRawArguments([
        getCurrentTime(),
        0,
        addressToBytes32Address(contractAddress),
        methodId,
        ...args,
      ]),
    ])
  }
})
