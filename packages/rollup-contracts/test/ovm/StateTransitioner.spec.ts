import '../setup'

/* External Imports */
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Logging */
const log = getLogger('state-transitioner', true)

/* Contract Imports */
import * as StateTransitioner from '../../build/StateTransitioner.json'
import * as PartialStateManager from '../../build/PartialStateManager.json'

/* Begin tests */
describe.only('StateTransitioner', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let stateTransitioner

  /* Deploy contracts before tests */
  beforeEach(async () => {
    stateTransitioner = await deployContract(wallet, StateTransitioner, [
      10, // Some fake transition index
      '0x' + '00'.repeat(32), // Some fake state root
      '0x' + '00'.repeat(20) // Some fake execution manager address
    ])
  })

  describe('Initialization', async () => {
    it('sets the fraud verifier address to the deployer', async () => {
      const fraudVerifierAddress = await stateTransitioner.fraudVerifier()
      fraudVerifierAddress.should.equal(wallet.address)
    })
  })

  describe('Pre-Execution', async () => {
    it('proves contract inclusion which allows us to query the isVerifiedContract in the state manager', async () => {
      const ovmContractAddress = '0x' + '01'.repeat(20)
      const codeContractAddress = stateTransitioner.address
      await stateTransitioner.proveContractInclusion(ovmContractAddress, codeContractAddress, 5)
      const stateManager = new Contract(await stateTransitioner.stateManager(), PartialStateManager.abi, wallet)

      const isVerified = await stateManager.isVerifiedContract(ovmContractAddress)
      isVerified.should.equal(true)
    })

    it('proves storage slot inclusion (after contract inclusion) allows us to query the storage', async () => {
      // First prove the contract
      const ovmContractAddress = '0x' + '01'.repeat(20)
      const codeContractAddress = stateTransitioner.address
      await stateTransitioner.proveContractInclusion(ovmContractAddress, codeContractAddress, 5)
      const stateManager = new Contract(await stateTransitioner.stateManager(), PartialStateManager.abi, wallet)

      // Next prove the storage
      const storageSlot = '0x' + '01'.repeat(32)
      const storageValue = '0x' + '11'.repeat(32)
      await stateTransitioner.proveStorageSlotInclusion(ovmContractAddress, storageSlot, storageValue)

      const isVerified = await stateManager.isVerifiedStorage(ovmContractAddress, storageSlot)
      isVerified.should.equal(true)
    })
  })
})
