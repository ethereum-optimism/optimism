import '../setup'

/* External Imports */
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Logging */
const log = getLogger('state-transitioner', true)

/* Contract Imports */
import * as StateTransitioner from '../../build/StateTransitioner.json'

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
})
