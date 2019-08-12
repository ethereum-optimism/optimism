import chai = require('chai')
import {
  createMockProvider,
  deployContract,
  getWallets,
  solidity,
} from 'ethereum-waffle'
import * as AggregatorWithIPCreationProxy from '../build/AggregatorWithIPCreationProxy.json'
import * as AggregatorRegistry from '../build/AggregatorRegistry.json'

chai.use(solidity)
const { expect } = chai

describe('Creates Aggregator and checks that fields are properly assigned', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let aggregatorWithIPCreationProxy
  let aggregatorRegistry

  beforeEach(async () => {
    const authenticationAddress = await wallet.getAddress()
    aggregatorRegistry = await deployContract(wallet, AggregatorRegistry, [], {
      gasLimit: 6700000,
    })
    aggregatorWithIPCreationProxy = await deployContract(
      wallet,
      AggregatorWithIPCreationProxy,
      [
        aggregatorRegistry.address,
        authenticationAddress,
        authenticationAddress,
      ],
      {
        gasLimit: 6700000,
      }
    )
  })

  it('Successfully self destructs contract', async () => {
    aggregatorWithIPCreationProxy.deleteThisContract()
  })

  it('Check that the aggregator registry added the aggregator', async () => {
    expect(await aggregatorRegistry.aggregators().length).to.eq(1)
  })

  it('Check that the aggregator contract is deployed', async () => {
    // You can check that it’s deployed with getCode in ethers.js
    expect(await provider.getCode(aggregatorRegistry.aggregators().get(0))).to
      .exist
  })

})
