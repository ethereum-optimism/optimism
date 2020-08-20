import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'
import { MessageChannel } from 'worker_threads'
import { initial } from 'lodash'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Tests */
describe('DepositedERC20', () => {
  const provider = ethers.provider

  let wallet: Signer
  let badwallet: Signer
  let userWallet: Signer
  let DepositedERC20: ContractFactory
  before(async () => {
    ;[wallet, badwallet, userWallet] = await ethers.getSigners()
    DepositedERC20 = await ethers.getContractFactory('DepositedERC20')
  })

  let depositedERC20: Contract
  beforeEach(async () => {
    depositedERC20 = await DepositedERC20.deploy(
      0,
      '_tokenName',
      10,
      '_tokenSymbol'
    )
  })

  describe('constructor()', async () => {
    it('sets DepositedERC20 factory address correctly', async () => {
      const factoryAddress = await depositedERC20.l2ERC20Bridge()
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
          await evilDepositedERC20.processDeposit('0x' + '00'.repeat(20), 5)
        }
      )
    })

    it('does not throw error if called by L2ERC20Bridge address', async () => {
      await depositedERC20.processDeposit('0x' + '00'.repeat(20), 5)
    })

    it('mints tokens and increases total supply', async () => {
      const initialTotalSupply = (await depositedERC20.totalSupply()).toNumber()
      const depositAmount = 5
      await depositedERC20.processDeposit('0x' + '00'.repeat(20), depositAmount)
      const newTotalSupply = (await depositedERC20.totalSupply()).toNumber()
      newTotalSupply.should.equal(initialTotalSupply + depositAmount)
    })

    it('mints tokens to the depositer', async () => {
      const depositerAddress = '0x' + '00'.repeat(20)
      const initialBalance = (
        await depositedERC20.balanceOf(depositerAddress)
      ).toNumber()
      const depositAmount = 5
      await depositedERC20.processDeposit('0x' + '00'.repeat(20), depositAmount)
      const newBalance = (
        await depositedERC20.balanceOf(depositerAddress)
      ).toNumber()
      newBalance.should.equal(initialBalance + depositAmount)
    })
  })

  describe('initializeWithdrawal()', async () => {
    it('burns tokens from message sender and decreases total supply', async () => {
      //Give wallet a balance to in order to withdraw
      await depositedERC20.processDeposit(userWallet.getAddress(), 100)
      const initialSupply = (await depositedERC20.totalSupply()).toNumber()
      const initialBalance = (
        await depositedERC20.balanceOf(userWallet.getAddress())
      ).toNumber()
      const withdrawalAmount = 5
      await depositedERC20
        .connect(userWallet)
        .initializeWithdrawal('0x' + '00'.repeat(20), withdrawalAmount)
      const newSupply = (await depositedERC20.totalSupply()).toNumber()
      const newBalance = (
        await depositedERC20.balanceOf(userWallet.getAddress())
      ).toNumber()
      // Tests that tokens are burned from withdrawer
      newBalance.should.equal(initialBalance - withdrawalAmount)
      // Tests that total token supply decreases by the same amount
      newSupply.should.equal(initialSupply - withdrawalAmount)
    })
  })
})
