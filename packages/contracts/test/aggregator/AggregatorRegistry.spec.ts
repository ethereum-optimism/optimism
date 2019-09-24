import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Contract Imports */
import * as AggregatorRegistry from '../../build/AggregatorRegistry.json'

/* Begin tests */
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
