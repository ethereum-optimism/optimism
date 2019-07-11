import chai = require('chai')
import {
  createMockProvider,
  deployContract,
  getWallets,
  solidity,
} from 'ethereum-waffle'
import * as AggregatorWithIPCreationProxy from '../build/AggregatorWithIPCreationProxy.json'
import * as PlasmaRegistry from '../build/PlasmaRegistry.json'

chai.use(solidity)
const { expect } = chai

describe('Creates Aggregator and checks that fields are properly assigned', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let aggregatorWithIPCreationProxy
  let plasmaRegistry

  beforeEach(async () => {
    const authenticationAddress = await wallet.getAddress()
    plasmaRegistry = await deployContract(wallet, PlasmaRegistry, [], {
      gasLimit: 6700000,
    })
    aggregatorWithIPCreationProxy = await deployContract(
      wallet,
      AggregatorWithIPCreationProxy,
      [plasmaRegistry.address, authenticationAddress, authenticationAddress],
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
