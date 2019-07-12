import chai = require('chai')
chai.use(require('chai-bignumber')())

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
  const [wallet1, wallet2] = getWallets(provider)
  let plasmaRegistry
  let agg1AuthenticationAddress
  let agg2AuthenticationAddress
  let agg1
  let agg2

  beforeEach(async () => {
    agg1AuthenticationAddress = await wallet1.getAddress()
    agg2AuthenticationAddress = await wallet2.getAddress()
    plasmaRegistry = await deployContract(wallet1, PlasmaRegistry, [], {
      gasLimit: 6700000,
    })
    agg1 = await plasmaRegistry.addAggregator(agg1AuthenticationAddress)
  })

  it('Creates aggregators and gives correct length ', async () => {
    let aggregatorCount = await plasmaRegistry.getAggregatorCount()
    aggregatorCount.should.be.bignumber.equal(1)
    agg2 = await plasmaRegistry.addAggregator(agg2AuthenticationAddress)
    aggregatorCount = await plasmaRegistry.getAggregatorCount()
    aggregatorCount.should.be.bignumber.equal(2)
  })
})
