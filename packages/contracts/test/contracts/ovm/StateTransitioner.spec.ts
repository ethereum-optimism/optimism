import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger } from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
} from '../../test-helpers'

/* Logging */
const log = getLogger('state-transitioner', true)

/* Contract Imports */
import * as PartialStateManager from '../../../artifacts/PartialStateManager.json'

/* Begin tests */
describe('StateTransitioner', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let StubExecutionManager: ContractFactory
  let StateTransitioner: ContractFactory
  before(async () => {
    StubExecutionManager = await ethers.getContractFactory(
      'StubExecutionManager'
    )
    StateTransitioner = await ethers.getContractFactory('StateTransitioner')
  })

  let stubExecutionManager: Contract
  let stateTransitioner: Contract
  beforeEach(async () => {
    stubExecutionManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: StubExecutionManager,
        params: [],
      }
    )

    stateTransitioner = await StateTransitioner.deploy(
      resolver.addressResolver.address,
      10,
      '0x' + '00'.repeat(32),
      stubExecutionManager.address
    )
  })

  const prepareStateForTransactionExecution = async () => {
    const contract1 = '0x' + '11'.repeat(20)
    const storageSlot1 = '0x' + '11'.repeat(32)
    const storageValue1 = '0x' + '11'.repeat(32)
    const contract2 = '0x' + '22'.repeat(20)
    const storageSlot2 = '0x' + '22'.repeat(32)
    const storageValue2 = '0x' + '22'.repeat(32)
    await stateTransitioner.proveContractInclusion(contract1, contract1, 1)
    await stateTransitioner.proveStorageSlotInclusion(
      contract1,
      storageSlot1,
      storageValue1
    )
    await stateTransitioner.proveContractInclusion(contract2, contract2, 5)
    await stateTransitioner.proveStorageSlotInclusion(
      contract2,
      storageSlot2,
      storageValue2
    )
  }

  describe('Initialization', async () => {
    it('sets the fraud verifier address to the deployer', async () => {
      const fraudVerifierAddress = await stateTransitioner.fraudVerifier()
      fraudVerifierAddress.should.equal(await wallet.getAddress())
    })
  })

  describe('Pre-Execution', async () => {
    it('proves contract inclusion which allows us to query the isVerifiedContract in the state manager', async () => {
      const ovmContractAddress = '0x' + '01'.repeat(20)
      const codeContractAddress = stateTransitioner.address
      await stateTransitioner.proveContractInclusion(
        ovmContractAddress,
        codeContractAddress,
        5
      )
      const stateManager = new Contract(
        await stateTransitioner.stateManager(),
        PartialStateManager.abi,
        wallet
      )

      const isVerified = await stateManager.isVerifiedContract(
        ovmContractAddress
      )
      isVerified.should.equal(true)
    })

    it('proves storage slot inclusion (after contract inclusion) allows us to query the storage', async () => {
      // First prove the contract
      const ovmContractAddress = '0x' + '01'.repeat(20)
      const codeContractAddress = stateTransitioner.address
      await stateTransitioner.proveContractInclusion(
        ovmContractAddress,
        codeContractAddress,
        5
      )
      const stateManager = new Contract(
        await stateTransitioner.stateManager(),
        PartialStateManager.abi,
        wallet
      )

      // Next prove the storage
      const storageSlot = '0x' + '01'.repeat(32)
      const storageValue = '0x' + '11'.repeat(32)
      await stateTransitioner.proveStorageSlotInclusion(
        ovmContractAddress,
        storageSlot,
        storageValue
      )

      const isVerified = await stateManager.isVerifiedStorage(
        ovmContractAddress,
        storageSlot
      )
      isVerified.should.equal(true)
    })
  })

  describe('applyTransaction(...)', async () => {
    it('fails if there was no state that was supplied', async () => {
      let didFail = false
      try {
        await stateTransitioner.applyTransaction()
      } catch (e) {
        didFail = true
      }
      didFail.should.equal(true)
    })

    it('does not fail if all the state is supplied', async () => {
      await prepareStateForTransactionExecution()
      await stateTransitioner.applyTransaction()
    })
  })
  describe('Post-Execution', async () => {
    it('moves between phases correctly', async () => {
      // TODO: Add real tests
      await prepareStateForTransactionExecution()
      await stateTransitioner.applyTransaction()
      await stateTransitioner.proveUpdatedStorageSlot()
      // Check that the phase is still post execution
      let phase = await stateTransitioner.currentTransitionPhase()
      phase.should.equal(1)
      await stateTransitioner.proveUpdatedStorageSlot()
      await stateTransitioner.completeTransition()
      phase = await stateTransitioner.currentTransitionPhase()
      // Check that the phase is now complete!
      phase.should.equal(2)
    })
  })
})
