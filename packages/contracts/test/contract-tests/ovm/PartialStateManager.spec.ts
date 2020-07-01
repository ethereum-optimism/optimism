import '../../setup'

/* External Imports */
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Logging */
const log = getLogger('partial-state-manager', true)

/* Contract Imports */
import * as PartialStateManager from '../../../build/contracts/PartialStateManager.json'

/* Begin tests */
describe('PartialStateManager', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let partialStateManager

  /* Deploy contracts before tests */
  beforeEach(async () => {
    partialStateManager = await deployContract(wallet, PartialStateManager, [
      wallet.address,
      wallet.address,
    ])
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
