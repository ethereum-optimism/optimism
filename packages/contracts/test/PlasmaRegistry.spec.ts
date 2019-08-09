import chai = require('chai')
import bignum = require('chai-bignumber')
chai.use(bignum())

import {
  createMockProvider,
  deployContract,
  getWallets,
  solidity,
} from 'ethereum-waffle'
import * as PlasmaRegistry from '../build/PlasmaRegistry.json'

chai.use(solidity)

describe('Creates Aggregator and checks that fields are properly assigned', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let plasmaRegistry
  let agg2AuthenticationAddress

  beforeEach(async () => {
    const agg1AuthenticationAddress = wallet1.address
    agg2AuthenticationAddress = wallet2.address
    plasmaRegistry = await deployContract(wallet1, PlasmaRegistry, [], {
      gasLimit: 6700000,
    })
    await plasmaRegistry.addAggregator(agg1AuthenticationAddress)
  })

  it('Creates aggregators and gives correct length ', async () => {
    let aggregatorCount = await plasmaRegistry.getAggregatorCount()
    aggregatorCount.should.be.bignumber.equal(1)
    await plasmaRegistry.addAggregator(agg2AuthenticationAddress)
    aggregatorCount = await plasmaRegistry.getAggregatorCount()
    aggregatorCount.should.be.bignumber.equal(2)
  })
})
