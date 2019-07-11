import chai = require('chai')
import {
  createMockProvider,
  deployContract,
  getWallets,
  solidity,
} from 'ethereum-waffle'
import * as PlasmaRegistry from '../build/PlasmaRegistry.json'

chai.use(solidity)
const { expect } = chai

describe('Creates Aggregator and checks that fields are properly assigned', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let plasmaRegistry
  let authenticationAddress

  beforeEach(async () => {
    authenticationAddress = await wallet.getAddress()
    plasmaRegistry = await deployContract(wallet, PlasmaRegistry, [], {
      gasLimit: 6700000,
    })
  })

  /**
   * AssertionError: expected {} to equal 0
   * at Context.it (test/PlasmaRegistry.spec.ts:21:46)
   * at process.internalTickCallback (internal/process/next_tick.js:77:7)
   * (node:15263) UnhandledPromiseRejectionWarning: TXRejectedError: the tx doesn't have the correct nonce. account has nonce of: 2 tx has nonce of: 1
   */
  it('assigns aggregators ', async () => {
    plasmaRegistry.addAggregator(authenticationAddress)
    expect(plasmaRegistry.getAggregatorCount()).to.eq(0)
  })
})
