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

  it('it deploys ', async () => {
    plasmaRegistry = await deployContract(wallet, PlasmaRegistry, [], {
      gasLimit: 6700000,
    })
  })
})
