import chai = require('chai')
import {
  createMockProvider,
  deployContract,
  getWallets,
  solidity,
} from 'ethereum-waffle'

import * as Aggregator from '../build/Aggregator.json'
import * as BasicTokenMock from '../build/BasicTokenMock.json'
import * as DummyDeposit from '../build/DummyDeposit.json'

chai.use(solidity)
const { expect } = chai

describe('Creates Aggregator and checks that fields are properly assigned', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let aggregator
  let depositContract
  let token

  beforeEach(async () => {
    const authenticationAddress = wallet.address
    const id = 121
    aggregator = await deployContract(
      wallet,
      Aggregator,
      [authenticationAddress, id],
      {
        gasLimit: 6700000,
      }
    )
  })

  it('Assigns AuthenticationAddress to Aggregator', async () => {
    expect(await aggregator.authenticationAddress()).to.eq(wallet.address)
  })

  it('Creates Commitment Chain', async () => {
    expect(await aggregator.commitmentContract()).to.exist
  })

  it('Assigns ID to Aggregator', async () => {
    expect(await aggregator.id()).to.eq(121)
  })

  it('Assigns and deletes IP in Metadata', async () => {
    const addr = wallet.address
    await aggregator.setMetadata(addr, 'heyo')
    expect(await aggregator.metadata(addr)).to.eq('heyo')
    await aggregator.deleteMetadata(addr)
    expect(await aggregator.metadata(addr)).to.eq('')
  })
})
