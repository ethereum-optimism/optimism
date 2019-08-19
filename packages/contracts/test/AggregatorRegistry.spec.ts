import chai = require('chai')
import bignum = require('chai-bignumber')
chai.use(bignum())

import {
  createMockProvider,
  deployContract,
  getWallets,
  solidity,
} from 'ethereum-waffle'
import * as AggregatorRegistry from '../build/AggregatorRegistry.json'

chai.use(solidity)

describe('AggregatorRegistry', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let aggregatorRegistry
  let agg2AuthenticationAddress

  beforeEach(async () => {
    const agg1AuthenticationAddress = wallet1.address
    agg2AuthenticationAddress = wallet2.address
    aggregatorRegistry = await deployContract(wallet1, AggregatorRegistry, [], {
      gasLimit: 6700000,
    })
    await aggregatorRegistry.addAggregator(agg1AuthenticationAddress)
  })

  it('getAggregatorCount() ', async () => {
    let aggregatorCount = await aggregatorRegistry.getAggregatorCount()
    aggregatorCount.should.be.bignumber.equal(1)
    await aggregatorRegistry.addAggregator(agg2AuthenticationAddress)
    aggregatorCount = await aggregatorRegistry.getAggregatorCount()
    aggregatorCount.should.be.bignumber.equal(2)
  })
})
