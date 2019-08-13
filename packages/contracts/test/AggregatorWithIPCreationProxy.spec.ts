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
  let authenticationAddress

  beforeEach(async () => {
    authenticationAddress = await wallet.getAddress()
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
    expect(await provider.getCode(aggregatorWithIPCreationProxy.address)).to
      .exist
    await aggregatorWithIPCreationProxy.deleteThisContract()
    expect(await provider.getCode(aggregatorWithIPCreationProxy.address)).to.eq(
      '0x'
    )
  })

  it('Check that the aggregator registry added the aggregator', async () => {
    const aggregatorCount = await aggregatorRegistry.getAggregatorCount()
    aggregatorCount.should.be.bignumber.equal(1)
  })

  it('Check that the aggregator contract is deployed', async () => {
    expect(await provider.getCode(await aggregatorRegistry.aggregators(0))).to
      .exist
  })
})
