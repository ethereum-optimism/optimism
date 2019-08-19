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

describe('AggregatorWithIPCreationProxy', () => {
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

  it('deleteThisContract()', async () => {
    expect(await provider.getCode(aggregatorWithIPCreationProxy.address)).to
      .exist
    await aggregatorWithIPCreationProxy.deleteThisContract()
    expect(await provider.getCode(aggregatorWithIPCreationProxy.address)).to.eq(
      '0x'
    )
  })

  it('aggregatorRegistry.getAggregatorCount()', async () => {
    const aggregatorCount = await aggregatorRegistry.getAggregatorCount()
    aggregatorCount.should.be.bignumber.equal(1)
  })

  it('aggregatorRegistry.aggregators()', async () => {
    expect(await provider.getCode(await aggregatorRegistry.aggregators(0))).to
      .exist
  })
})
