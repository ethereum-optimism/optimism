import chai = require('chai')
import {
  createMockProvider,
  deployContract,
  getWallets,
  solidity,
} from 'ethereum-waffle'
import * as AggregatorWithIPCreationProxy from '../build/AggregatorWithIPCreationProxy.json'

chai.use(solidity)
const { expect } = chai

describe('Creates Aggregator and checks that fields are properly assigned', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let aggregatorWithIPCreationProxy

  beforeEach(async () => {
    const authenticationAddress = await wallet.getAddress()
    aggregatorWithIPCreationProxy = await deployContract(
      wallet,
      AggregatorWithIPCreationProxy,
      [authenticationAddress, authenticationAddress],
      {
        gasLimit: 6700000,
      }
    )
  })

  it('Successfully self destructs contract', async () => {
    aggregatorWithIPCreationProxy.deleteThisContract()
    expect(aggregatorWithIPCreationProxy.owner()).to.be.empty
  })
})
