import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  remove0x,
  TestUtils,
  ZERO_ADDRESS,
  NULL_ADDRESS,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  GAS_LIMIT,
  OVM_METHOD_IDS,
  Address,
  manuallyDeployOvmContract,
  addressToBytes32Address,
  didCreateSucceed,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  executeTestTransaction,
  executePersistedTestTransaction,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-calls', true)

const sloadKey: string = '11'.repeat(32)
const unpopultedSLOADResult: string = '00'.repeat(32)
const populatedSLOADResult: string = '22'.repeat(32)

/* Tests */
describe('Execution Manager -- Call opcodes', () => {
  const provider = ethers.provider

  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let DummyContract: ContractFactory
  let dummyContract: Contract
  let SimpleCall: ContractFactory
  let deployTx: any
  before(async () => {
    DummyContract = await ethers.getContractFactory('DummyContract')
    dummyContract = await DummyContract.deploy()

    SimpleCall = await ethers.getContractFactory('SimpleCall')
    deployTx = SimpleCall.getDeployTransaction(dummyContract.address)
  })

  let ExecutionManager: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
  })

  let executionManager: Contract
  beforeEach(async () => {
    executionManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: ExecutionManager,
        params: [resolver.addressResolver.address, NULL_ADDRESS, GAS_LIMIT],
      }
    )
  })

  let callContractAddress: Address
  let callContract2Address: Address
  let callContract3Address: Address
  beforeEach(async () => {
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
    it('properly executes ovmCALL to SLOAD', async () => {
      const result: string = await executeTestTransaction(
        executionManager,
        callContractAddress,
        'staticFriendlySLOAD',
        [sloadKey]
      )
      log.debug(`Result: [${result}]`)

      remove0x(result).should.equal(unpopultedSLOADResult, 'Result mismatch!')
    })

    it('properly executes ovmCALL to SSTORE', async () => {
      await executePersistedTestTransaction(
        executionManager,
        wallet,
        callContractAddress,
        'makeCall',
        [
          addressToBytes32Address(callContract2Address),
          OVM_METHOD_IDS.notStaticFriendlySSTORE,
          sloadKey,
          populatedSLOADResult,
        ]
      )

      const result: string = await executeTestTransaction(
        executionManager,
        callContract2Address,
        'staticFriendlySLOAD',
        [sloadKey]
      )

      log.debug(`Result: [${result}]`)

      // Stored in contract 2, matches contract 2
      remove0x(result).should.equal(populatedSLOADResult, 'SLOAD mismatch!')
    })

    it('properly executes ovmCALL to CREATE', async () => {
      const result: string = await executeTestTransaction(
        executionManager,
        callContract2Address,
        'notStaticFriendlyCREATE',
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
      const result: string = await executeTestTransaction(
        executionManager,
        callContract2Address,
        'notStaticFriendlyCREATE2',
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
      await executePersistedTestTransaction(
        executionManager,
        wallet,
        callContractAddress,
        'makeDelegateCall',
        [
          addressToBytes32Address(callContract2Address),
          OVM_METHOD_IDS.notStaticFriendlySSTORE,
          sloadKey,
          populatedSLOADResult,
        ]
      )

      // Stored in contract 2 via delegate call but accessed via contract 1
      const result: string = await executeTestTransaction(
        executionManager,
        callContractAddress,
        'staticFriendlySLOAD',
        [sloadKey]
      )

      log.debug(`Result: [${result}]`)
      // Should have stored result
      remove0x(result).should.equal(
        populatedSLOADResult,
        'SLOAD should yield stored result!'
      )

      const contract2Result: string = await executeTestTransaction(
        executionManager,
        callContract2Address,
        'staticFriendlySLOAD',
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
      const result = await executePersistedTestTransaction(
        executionManager,
        wallet,
        callContractAddress,
        'makeDelegateCall',
        [
          addressToBytes32Address(callContract2Address),
          OVM_METHOD_IDS.makeDelegateCall,
          addressToBytes32Address(callContract3Address),
          OVM_METHOD_IDS.notStaticFriendlySSTORE,
          sloadKey,
          populatedSLOADResult,
        ]
      )

      const contract1Result: string = await executeTestTransaction(
        executionManager,
        callContractAddress,
        'staticFriendlySLOAD',
        [sloadKey]
      )

      log.debug(`Result 1: [${contract1Result}]`)

      // Stored in contract 3 via delegate call but accessed via contract 1
      remove0x(contract1Result).should.equal(
        populatedSLOADResult,
        'SLOAD should yield stored data!'
      )

      const contract2Result: string = await executeTestTransaction(
        executionManager,
        callContract2Address,
        'staticFriendlySLOAD',
        [sloadKey]
      )

      log.debug(`Result 2: [${contract2Result}]`)

      // Should not be stored
      remove0x(contract2Result).should.equal(
        unpopultedSLOADResult,
        'SLOAD should not yield any data (0 x 32 bytes)!'
      )

      const contract3Result: string = await executeTestTransaction(
        executionManager,
        callContract3Address,
        'staticFriendlySLOAD',
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
      const result = await executeTestTransaction(
        executionManager,
        callContractAddress,
        'makeStaticCall',
        [
          addressToBytes32Address(callContract2Address),
          OVM_METHOD_IDS.staticFriendlySLOAD,
          sloadKey,
        ]
      )

      log.debug(`Result: [${result}]`)

      remove0x(result).should.equal(unpopultedSLOADResult, 'Result mismatch!')
    })

    it('properly executes nested ovmSTATICCALL to SLOAD', async () => {
      const result = await executeTestTransaction(
        executionManager,
        callContractAddress,
        'makeStaticCall',
        [
          addressToBytes32Address(callContract2Address),
          OVM_METHOD_IDS.makeStaticCall,
          addressToBytes32Address(callContract2Address),
          OVM_METHOD_IDS.staticFriendlySLOAD,
          sloadKey,
        ]
      )

      log.debug(`Result: [${result}]`)

      remove0x(result).should.equal(unpopultedSLOADResult, 'Result mismatch!')
    })

    it('successfully makes static call then call', async () => {
      // Should not throw
      await executeTestTransaction(
        executionManager,
        callContractAddress,
        'makeStaticCallThenCall',
        [addressToBytes32Address(callContractAddress)]
      )
    })

    it('remains in static context when exiting nested static context', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executePersistedTestTransaction(
          executionManager,
          wallet,
          callContractAddress,
          'makeStaticCall',
          [
            addressToBytes32Address(callContractAddress),
            OVM_METHOD_IDS.makeStaticCallThenCall,
            addressToBytes32Address(callContractAddress),
          ]
        )
      })
    })

    it('fails on ovmSTATICCALL to SSTORE', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executePersistedTestTransaction(
          executionManager,
          wallet,
          callContractAddress,
          'makeStaticCall',
          [
            addressToBytes32Address(callContractAddress),
            OVM_METHOD_IDS.notStaticFriendlySSTORE,
            sloadKey,
            populatedSLOADResult,
          ]
        )
      })
    })

    it('Fails to create on ovmSTATICCALL to CREATE -- tx', async () => {
      const hash = await executePersistedTestTransaction(
        executionManager,
        wallet,
        callContractAddress,
        'makeStaticCall',
        [
          addressToBytes32Address(callContractAddress),
          OVM_METHOD_IDS.notStaticFriendlyCREATE,
          deployTx.data,
        ]
      )
      const createSucceeded = await didCreateSucceed(executionManager, hash)

      createSucceeded.should.equal(false, 'Create should have failed!')
    })

    it('Fails to create on ovmSTATICCALL to CREATE -- call', async () => {
      const address = await executeTestTransaction(
        executionManager,
        callContractAddress,
        'makeStaticCall',
        [
          addressToBytes32Address(callContractAddress),
          OVM_METHOD_IDS.notStaticFriendlyCREATE,
          deployTx.data,
        ]
      )

      address.should.equal(
        addressToBytes32Address(ZERO_ADDRESS),
        'Should be 0 address!'
      )
    })

    it('fails on ovmSTATICCALL to CREATE2 -- tx', async () => {
      const hash = await executePersistedTestTransaction(
        executionManager,
        wallet,
        callContractAddress,
        'makeStaticCall',
        [
          addressToBytes32Address(callContractAddress),
          OVM_METHOD_IDS.notStaticFriendlyCREATE2,
          0,
          deployTx.data,
        ]
      )

      const createSucceeded = await didCreateSucceed(executionManager, hash)
      createSucceeded.should.equal(false, 'Create should have failed!')
    })

    it('fails on ovmSTATICCALL to CREATE2 -- call', async () => {
      const res = await executeTestTransaction(
        executionManager,
        callContractAddress,
        'makeStaticCall',
        [
          addressToBytes32Address(callContractAddress),
          OVM_METHOD_IDS.notStaticFriendlyCREATE2,
          0,
          deployTx.data,
        ]
      )

      res.should.equal(
        addressToBytes32Address(ZERO_ADDRESS),
        'Should be 0 address!'
      )
    })
  })
})
