import '../../common/setup'

/* External Imports */
import { link, deployContract } from 'ethereum-waffle'
import { Wallet, Contract } from 'ethers'

/* Internal Imports */
import { waffle } from '../../../src/waffle'

/* Contract Imports */
import * as SimpleSafeMathJSON from '../../temp/build/waffle/SimpleSafeMath.json'
import * as SimpleUnsafeMathJSON from '../../temp/build/waffle/SimpleUnsafeMath.json'
import * as SafeMathUserJSON from '../../temp/build/waffle/SafeMathUser.json'

const CONTRACT_PATH_PREFIX = 'test/common/contracts/libraries/'

describe('Library Support', () => {
  let provider: any
  let wallet: Wallet
  before(async () => {
    provider = new waffle.MockProvider()
    ;[wallet] = provider.getWallets()
  })

  let deployedLibUser: Contract
  before(async () => {
    const deployedSafeMath = await deployContract(
      wallet,
      SimpleSafeMathJSON,
      []
    )
    link(
      SafeMathUserJSON,
      CONTRACT_PATH_PREFIX + 'SimpleSafeMath.sol:SimpleSafeMath',
      deployedSafeMath.address
    )

    const deployedUnsafeMath = await deployContract(
      wallet,
      SimpleUnsafeMathJSON,
      []
    )
    link(
      SafeMathUserJSON,
      CONTRACT_PATH_PREFIX + 'SimpleUnsafeMath.sol:SimpleUnsafeMath',
      deployedUnsafeMath.address
    )

    deployedLibUser = await deployContract(wallet, SafeMathUserJSON, [])
  })

  it('should allow us to transpile, link, and query contract methods which use a single library', async () => {
    const returnedUsingLib = await deployedLibUser.useLib()
    returnedUsingLib._hex.should.equal('0x05')
  }).timeout(20_000)

  it('should allow us to transpile, link, and query contract methods which use a multiple libraries', async () => {
    const returnedUsingLib = await deployedLibUser.use2Libs()
    returnedUsingLib._hex.should.equal('0x06')
  }).timeout(20_000)
})
