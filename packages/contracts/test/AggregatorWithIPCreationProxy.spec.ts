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

  it('it deploys ', async () => {
    aggregatorWithIPCreationProxy = await deployContract(
      wallet,
      AggregatorWithIPCreationProxy,
      [],
      {
        gasLimit: 6700000,
      }
    )
  })
})
