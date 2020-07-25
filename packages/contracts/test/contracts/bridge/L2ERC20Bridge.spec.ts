import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Tests */
describe('L2ERC20Bridge', () => {
  const provider = ethers.provider

  let wallet: Signer
  let L2ERC20Bridge: ContractFactory
  before(async () => {
    ;[wallet] = await ethers.getSigners()
    L2ERC20Bridge = await ethers.getContractFactory('L2ERC20Bridge')
  })

  let l2ERC20Bridge: Contract
  beforeEach(async () => {
    l2ERC20Bridge = await L2ERC20Bridge.deploy()
  })

  describe.only('constructor()', async () => {
    it('Sets L2ERC20Bridge factory address correctly', async () => {
      const factoryAddress = await l2ERC20Bridge.l2BridgeFactoryAddress()
      factoryAddress.should.equal(await wallet.getAddress())
      console.log('test')
    })
  })
})
