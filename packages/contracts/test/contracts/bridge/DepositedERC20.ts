import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Tests */
describe.only('DepositedERC20', () => {
  const provider = ethers.provider

  let wallet: Signer
  let badwallet: Signer
  let DepositedERC20: ContractFactory
  before(async () => {
    ;[wallet, badwallet] = await ethers.getSigners()
    DepositedERC20 = await ethers.getContractFactory('DepositedERC20')
  })

  let depositedERC20: Contract
  beforeEach(async () => {
    depositedERC20 = await DepositedERC20.deploy()
  })

  describe('constructor()', async () => {
    it('sets DepositedERC20 factory address correctly', async () => {
      const factoryAddress = await depositedERC20.l2ERC20BridgeAddress()
      factoryAddress.should.equal(await wallet.getAddress())
      console.log('test')
    })
  })

  describe('processDeposit()', async () => {
    it('throws error if msg sender is not L2ERC20Bridge address', async () => {
      const evilDepositedERC20 = depositedERC20.connect(badwallet)
      await TestUtils.assertRevertsAsync(
        'Get outta here. L2 factory bridge address ONLY.',
        async () => {
          await evilDepositedERC20.processDeposit("0x"+"00".repeat(20), 5)
        }
      )
    })
  })

})
