import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  remove0x,
  TestUtils,
  NULL_ADDRESS,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT,
  executeOVMCall,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  encodeFunctionData,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-create', true)

/* Tests */
describe('ExecutionManager -- Create opcodes', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let ExecutionManager: ContractFactory
  let SafetyChecker: ContractFactory
  let StubSafetyChecker: ContractFactory
  let SimpleStorage: ContractFactory
  let InvalidOpcodes: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    SafetyChecker = await ethers.getContractFactory('SafetyChecker')
    StubSafetyChecker = await ethers.getContractFactory('StubSafetyChecker')
    SimpleStorage = await ethers.getContractFactory('SimpleStorage')
    InvalidOpcodes = await ethers.getContractFactory('InvalidOpcodes')
  })

  let safetyChecker: Contract
  let stubSafetyChecker: Contract
  before(async () => {
    safetyChecker = await SafetyChecker.deploy(
      resolver.addressResolver.address,
      DEFAULT_OPCODE_WHITELIST_MASK
    )
    stubSafetyChecker = await StubSafetyChecker.deploy()
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

  let deployTx: any
  let deployInvalidTx: any
  beforeEach(async () => {
    deployTx = SimpleStorage.getDeployTransaction()
    deployInvalidTx = InvalidOpcodes.getDeployTransaction()
  })

  describe('ovmCREATE', async () => {
    it('returns created address when passed valid bytecode', async () => {
      await resolver.addressResolver.setAddress(
        'SafetyChecker',
        stubSafetyChecker.address
      )

      const result = await executeOVMCall(executionManager, 'ovmCREATE', [
        deployTx.data,
      ])

      log.debug(`Result: [${result}]`)

      const address: string = remove0x(result)
      address.length.should.equal(64, 'Should be a full word for the address')
      address.should.not.equal('00'.repeat(32), 'Should not be 0 address')
    })

    it('reverts when passed unsafe bytecode', async () => {
      await resolver.addressResolver.setAddress(
        'SafetyChecker',
        safetyChecker.address
      )

      const data = encodeFunctionData('ovmCREATE', [deployInvalidTx.data])

      await TestUtils.assertRevertsAsync(
        'Contract init (creation) code is not safe',
        async () => {
          await executionManager.provider.call({
            to: executionManager.address,
            data,
            gasLimit: GAS_LIMIT,
          })
        }
      )
    })
  })

  describe('ovmCREATE2', async () => {
    it('returns created address when passed salt and bytecode', async () => {
      await resolver.addressResolver.setAddress(
        'SafetyChecker',
        stubSafetyChecker.address
      )

      const data = encodeFunctionData('ovmCREATE2', [0, deployTx.data])

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: GAS_LIMIT,
      })

      log.debug(`Result: [${result}]`)

      const address: string = remove0x(result)
      address.length.should.equal(64, 'Should be a full word for the address')
      address.should.not.equal('00'.repeat(32), 'Should not be 0 address')
    })

    it('reverts when passed unsafe bytecode', async () => {
      await resolver.addressResolver.setAddress(
        'SafetyChecker',
        safetyChecker.address
      )

      const data = encodeFunctionData('ovmCREATE2', [0, deployInvalidTx.data])

      await TestUtils.assertRevertsAsync(
        'Contract init (creation) code is not safe',
        async () => {
          await executionManager.provider.call({
            to: executionManager.address,
            data,
            gasLimit: GAS_LIMIT,
          })
        }
      )
    })
  })
})
