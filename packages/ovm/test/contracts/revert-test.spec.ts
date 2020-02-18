import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Contract Imports */
import * as RevertTest from '../../build/contracts/RevertTest.json'

/* Internal Imports */
import { DEFAULT_ETHNODE_GAS_LIMIT } from '../helpers'

/*********
 * TESTS *
 *********/

describe('Revert Test', () => {
  const [wallet] = getWallets(
    createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  )
  let revertTestContract

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // First deploy the contract address
    revertTestContract = await deployContract(wallet, RevertTest, [], {
      gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
    })
  })

  describe('Test that revert will blow away modified state from successful sub-calls', async () => {
    it.only('reverts sub-call state', async () => {
      await revertTestContract.entryPoint()

      const a = await revertTestContract.getA()
      a.should.equal(
        0,
        'Revert should revert the state update of a = 0 -> a = 5'
      )
    })
  })
})
