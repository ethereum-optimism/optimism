import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Logging */
const log = getLogger('partial-state-manager', true)

/* Begin tests */
describe('PartialStateManager', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let PartialStateManager: ContractFactory
  before(async () => {
    PartialStateManager = await ethers.getContractFactory('PartialStateManager')
  })

  let partialStateManager
  beforeEach(async () => {
    partialStateManager = await PartialStateManager.deploy(
      await wallet.getAddress(),
      await wallet.getAddress()
    )
  })

  describe('Pre-Execution', async () => {
    describe('Storage Verification', async () => {
      it('does not set existsInvalidStateAccessFlag=true if getStorage(contract, key) is called with a verified value', async () => {
        const address = '0x' + '01'.repeat(20)
        const key = '0x' + '01'.repeat(32)
        const value = '0x' + '01'.repeat(32)

        // First verify the value
        await partialStateManager.insertVerifiedStorage(address, key, value)
        // Then access
        await partialStateManager.getStorage(address, key)

        const existsInvalidStateAccessFlag = await partialStateManager.existsInvalidStateAccessFlag()
        existsInvalidStateAccessFlag.should.equal(false)
      })

      it('sets existsInvalidStateAccessFlag=true if getStorage(contract, key) is called without being verified', async () => {
        const address = '0x' + '01'.repeat(20)
        const key = '0x' + '01'.repeat(32)

        // Attempt to get unverified storage!
        await partialStateManager.getStorage(address, key)

        const existsInvalidStateAccessFlag = await partialStateManager.existsInvalidStateAccessFlag()
        existsInvalidStateAccessFlag.should.equal(true)
      })

      it('sets existsInvalidStateAccessFlag=true if setStorage(contract, key, value) is called without being verified', async () => {
        const address = '0x' + '01'.repeat(20)
        const key = '0x' + '01'.repeat(32)
        const value = '0x' + '01'.repeat(32)

        // Attempt to set unverified storage!
        await partialStateManager.setStorage(address, key, value)

        const existsInvalidStateAccessFlag = await partialStateManager.existsInvalidStateAccessFlag()
        existsInvalidStateAccessFlag.should.equal(true)
      })
    })

    describe('Contract Verification', async () => {
      // TODO
    })
  })
  describe('Post-Execution', async () => {
    // TODO
  })
})
