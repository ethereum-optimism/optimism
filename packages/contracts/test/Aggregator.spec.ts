import chai = require('chai')
import {
  createMockProvider,
  deployContract,
  getWallets,
  solidity,
} from 'ethereum-waffle'
import * as Aggregator from '../build/Aggregator.json'

chai.use(solidity)
const { expect } = chai

describe('Creates Aggregator and checks that fields are properly assigned', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let aggregator

  it('it deploys ', async () => {
    aggregator = await deployContract(wallet, Aggregator, [], {
      gasLimit: 6700000,
    })
  })
})
