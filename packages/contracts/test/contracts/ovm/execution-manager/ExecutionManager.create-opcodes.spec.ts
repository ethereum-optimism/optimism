import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  remove0x,
  add0x,
  TestUtils,
  NULL_ADDRESS,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer, providers, Wallet } from 'ethers'
import { fromPairs } from 'lodash'

/* Internal Imports */
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT,
  executeOVMCall,
  encodeMethodId,
  encodeRawArguments,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  getDefaultGasMeterParams,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-create', true)

const methodIds = fromPairs(
  ['ovmCREATE', 'ovmCREATE2'].map((methodId) => [
    methodId,
    encodeMethodId(methodId),
  ])
)

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
        params: [
          resolver.addressResolver.address,
          NULL_ADDRESS,
          getDefaultGasMeterParams(),
        ],
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

      const data = add0x(
        methodIds.ovmCREATE + encodeRawArguments([deployInvalidTx.data])
      )
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
    it('returns 00000s when attempting to redeploy with same nonce', async () => {
      // first actually CREATE
      const tx = {
        to: executionManager.address,
        data: add0x(
          methodIds.ovmCREATE + encodeRawArguments([deployTx.data])
        ),
        gasLimit: GAS_LIMIT,
      }
      await wallet.sendTransaction(tx)

      // reset nonce (contract is zero address since direct call to ovmCREATE does not initialize an active contract)
      await resolver.contracts.stateManager.setOvmContractNonce(ZERO_ADDRESS, 0)
      
      // now try to CREATE with same nonce
      const result = await executionManager.provider.call(tx)
      const address: string = remove0x(result)
      address.should.equal('00'.repeat(32), 'Should not be 0 address')
    })
  })

  describe('ovmCREATE2', async () => {
    it('returns created address when passed salt and bytecode', async () => {
      await resolver.addressResolver.setAddress(
        'SafetyChecker',
        stubSafetyChecker.address
      )

      const data = add0x(
        methodIds.ovmCREATE2 + encodeRawArguments([0, deployTx.data])
      )

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: GAS_LIMIT,
      })

      const address: string = remove0x(result)
      address.length.should.equal(64, 'Should be a full word for the address')
      address.should.not.equal('00'.repeat(32), 'Should not be 0 address')
    })

    it('returns 00000s when attempting to redeploy with same salt', async () => {
      await resolver.addressResolver.setAddress(
        'SafetyChecker',
        stubSafetyChecker.address
      )

      const data = add0x(
        methodIds.ovmCREATE2 + encodeRawArguments([0, deployTx.data])
      )

      const tx = {
        to: executionManager.address,
        data,
        gasLimit: GAS_LIMIT,
      }

      await wallet.sendTransaction(tx)

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: GAS_LIMIT,
      })

      const address: string = remove0x(result)
      address.should.equal('00'.repeat(32), 'Should not be 0 address')
    })

    it('reverts when passed unsafe bytecode', async () => {
      await resolver.addressResolver.setAddress(
        'SafetyChecker',
        safetyChecker.address
      )

      const data = add0x(
        methodIds.ovmCREATE2 + encodeRawArguments([0, deployInvalidTx.data])
      )

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
