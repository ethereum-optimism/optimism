import '../setup'

/* External Imports */
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Logging */
const log = getLogger('l1-to-l2-tx-queue', true)

/* Contract Imports */
import * as PartialStateManager from '../../build/PartialStateManager.json'
import * as StubSafetyChecker from '../../build/StubSafetyChecker.json'

/* Begin tests */
describe.only('PartialStateManager', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let stubSafetyChecker
  let partialStateManager

  before(async () => {
    stubSafetyChecker = await deployContract(wallet, StubSafetyChecker, [])
  })

  /* Link libraries before tests */
  beforeEach(async () => {
    partialStateManager = await deployContract(wallet, PartialStateManager, [stubSafetyChecker.address, wallet.address])
  })

  describe('setStorage() ', async () => {
    it('should not fail', async () => {
      console.log('success!')
    })
  })
})
