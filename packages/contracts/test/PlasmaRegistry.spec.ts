import chai = require('chai')
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
  const [wallet] = getWallets(provider)
  let plasmaRegistry
  let authenticationAddress

  beforeEach(async () => {
    authenticationAddress = await wallet.getAddress()
    plasmaRegistry = await deployContract(wallet, PlasmaRegistry, [], {
      gasLimit: 6700000,
    })
    plasmaRegistry.addAggregator(authenticationAddress)
  })
})
