import { expect } from 'chai'

/* Imports: External */
import { Wallet, utils, BigNumber } from 'ethers'
import { predeploys } from '@eth-optimism/contracts'

/* Imports: Internal */
import { Direction } from './shared/watcher-utils'

import {
  expectApprox,
  fundUser,
  PROXY_SEQUENCER_ENTRYPOINT_ADDRESS,
} from './shared/utils'
import { OptimismEnv, useDynamicTimeoutForWithdrawals } from './shared/env'

const DEFAULT_TEST_GAS_L1 = 330_000
const DEFAULT_TEST_GAS_L2 = 1_300_000
// TX size enforced by CTC:
const MAX_ROLLUP_TX_SIZE = 50_000

describe('Native ETH Integration Tests', async () => {
  let env: OptimismEnv
  let l1Bob: Wallet
  let l2Bob: Wallet

  const getBalances = async (_env: OptimismEnv) => {
    const l1UserBalance = await _env.l1Wallet.getBalance()
    const l2UserBalance = await _env.l2Wallet.getBalance()

    const l1BobBalance = await l1Bob.getBalance()
    const l2BobBalance = await l2Bob.getBalance()

    const sequencerBalance = await _env.ovmEth.balanceOf(
      PROXY_SEQUENCER_ENTRYPOINT_ADDRESS
    )
    const l1BridgeBalance = await _env.l1Wallet.provider.getBalance(
      _env.l1Bridge.address
    )

    return {
      l1UserBalance,
      l2UserBalance,
      l1BobBalance,
      l2BobBalance,
      l1BridgeBalance,
      sequencerBalance,
    }
  }

  before(async () => {
    env = await OptimismEnv.new()
    l1Bob = Wallet.createRandom().connect(env.l1Wallet.provider)
    l2Bob = l1Bob.connect(env.l2Wallet.provider)
  })

  describe('estimateGas', () => {
    it('Should estimate gas for ETH transfer', async () => {
      const amount = utils.parseEther('0.0000001')
      const addr = '0x' + '1234'.repeat(10)
      const gas = await env.ovmEth.estimateGas.transfer(addr, amount)
      // Expect gas to be less than or equal to the target plus 1%
      expectApprox(gas, 6430020, { upperPercentDeviation: 1 })
    })

    it('Should estimate gas for ETH withdraw', async () => {
      const amount = utils.parseEther('0.0000001')
      const gas = await env.l2Bridge.estimateGas.withdraw(
        predeploys.OVM_ETH,
        amount,
        0,
        '0xFFFF'
      )
      // Expect gas to be less than or equal to the target plus 1%
      expectApprox(gas, 6700060, { upperPercentDeviation: 1 })
    })
  })

  it('receive', async () => {
    const depositAmount = 10
    const preBalances = await getBalances(env)
    const { tx, receipt } = await env.waitForXDomainTransaction(
      env.l1Wallet.sendTransaction({
        to: env.l1Bridge.address,
        value: depositAmount,
        gasLimit: DEFAULT_TEST_GAS_L1,
      }),
      Direction.L1ToL2
    )

    const l1FeePaid = receipt.gasUsed.mul(tx.gasPrice)
    const postBalances = await getBalances(env)

    expect(postBalances.l1BridgeBalance).to.deep.eq(
      preBalances.l1BridgeBalance.add(depositAmount)
    )
    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.add(depositAmount)
    )
    expect(postBalances.l1UserBalance).to.deep.eq(
      preBalances.l1UserBalance.sub(l1FeePaid.add(depositAmount))
    )
  })

  it('depositETH', async () => {
    const depositAmount = 10
    const preBalances = await getBalances(env)
    const { tx, receipt } = await env.waitForXDomainTransaction(
      env.l1Bridge.depositETH(DEFAULT_TEST_GAS_L2, '0xFFFF', {
        value: depositAmount,
        gasLimit: DEFAULT_TEST_GAS_L1,
      }),
      Direction.L1ToL2
    )

    const l1FeePaid = receipt.gasUsed.mul(tx.gasPrice)
    const postBalances = await getBalances(env)

    expect(postBalances.l1BridgeBalance).to.deep.eq(
      preBalances.l1BridgeBalance.add(depositAmount)
    )
    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.add(depositAmount)
    )
    expect(postBalances.l1UserBalance).to.deep.eq(
      preBalances.l1UserBalance.sub(l1FeePaid.add(depositAmount))
    )
  })

  it('depositETHTo', async () => {
    const depositAmount = 10
    const preBalances = await getBalances(env)
    const depositReceipts = await env.waitForXDomainTransaction(
      env.l1Bridge.depositETHTo(l2Bob.address, DEFAULT_TEST_GAS_L2, '0xFFFF', {
        value: depositAmount,
        gasLimit: DEFAULT_TEST_GAS_L1,
      }),
      Direction.L1ToL2
    )

    const l1FeePaid = depositReceipts.receipt.gasUsed.mul(
      depositReceipts.tx.gasPrice
    )
    const postBalances = await getBalances(env)
    expect(postBalances.l1BridgeBalance).to.deep.eq(
      preBalances.l1BridgeBalance.add(depositAmount)
    )
    expect(postBalances.l2BobBalance).to.deep.eq(
      preBalances.l2BobBalance.add(depositAmount)
    )
    expect(postBalances.l1UserBalance).to.deep.eq(
      preBalances.l1UserBalance.sub(l1FeePaid.add(depositAmount))
    )
  })

  it('deposit passes with a large data argument', async () => {
    const ASSUMED_L2_GAS_LIMIT = 8_000_000
    const depositAmount = 10
    const preBalances = await getBalances(env)

    // Set data length slightly less than MAX_ROLLUP_TX_SIZE
    // to allow for encoding and other arguments
    const data = `0x` + 'ab'.repeat(MAX_ROLLUP_TX_SIZE - 500)
    const { tx, receipt } = await env.waitForXDomainTransaction(
      env.l1Bridge.depositETH(ASSUMED_L2_GAS_LIMIT, data, {
        value: depositAmount,
        gasLimit: 4_000_000,
      }),
      Direction.L1ToL2
    )

    const l1FeePaid = receipt.gasUsed.mul(tx.gasPrice)
    const postBalances = await getBalances(env)
    expect(postBalances.l1BridgeBalance).to.deep.eq(
      preBalances.l1BridgeBalance.add(depositAmount)
    )
    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.add(depositAmount)
    )
    expect(postBalances.l1UserBalance).to.deep.eq(
      preBalances.l1UserBalance.sub(l1FeePaid.add(depositAmount))
    )
  })

  it('depositETH fails with a TOO large data argument', async () => {
    const depositAmount = 10

    const data = `0x` + 'ab'.repeat(MAX_ROLLUP_TX_SIZE + 1)
    await expect(
      env.l1Bridge.depositETH(DEFAULT_TEST_GAS_L2, data, {
        value: depositAmount,
      })
    ).to.be.reverted
  })

  it('withdraw', async function () {
    await useDynamicTimeoutForWithdrawals(this, env)

    const withdrawAmount = BigNumber.from(3)
    const preBalances = await getBalances(env)
    expect(
      preBalances.l2UserBalance.gt(0),
      'Cannot run withdrawal test before any deposits...'
    )

    const transaction = await env.l2Bridge.withdraw(
      predeploys.OVM_ETH,
      withdrawAmount,
      DEFAULT_TEST_GAS_L2,
      '0xFFFF'
    )
    await transaction.wait()
    await env.relayXDomainMessages(transaction)
    const receipts = await env.waitForXDomainTransaction(
      transaction,
      Direction.L2ToL1
    )
    const fee = receipts.tx.gasLimit.mul(receipts.tx.gasPrice)

    const postBalances = await getBalances(env)

    // Approximate because there's a fee related to relaying the L2 => L1 message and it throws off the math.
    expectApprox(
      postBalances.l1BridgeBalance,
      preBalances.l1BridgeBalance.sub(withdrawAmount),
      { upperPercentDeviation: 1 }
    )
    expectApprox(
      postBalances.l2UserBalance,
      preBalances.l2UserBalance.sub(withdrawAmount.add(fee)),
      { upperPercentDeviation: 1 }
    )
    expectApprox(
      postBalances.l1UserBalance,
      preBalances.l1UserBalance.add(withdrawAmount),
      { upperPercentDeviation: 1 }
    )
  })

  it('withdrawTo', async function () {
    await useDynamicTimeoutForWithdrawals(this, env)

    const withdrawAmount = BigNumber.from(3)

    const preBalances = await getBalances(env)

    expect(
      preBalances.l2UserBalance.gt(0),
      'Cannot run withdrawal test before any deposits...'
    )

    const transaction = await env.l2Bridge.withdrawTo(
      predeploys.OVM_ETH,
      l1Bob.address,
      withdrawAmount,
      DEFAULT_TEST_GAS_L2,
      '0xFFFF'
    )
    await transaction.wait()
    await env.relayXDomainMessages(transaction)
    const receipts = await env.waitForXDomainTransaction(
      transaction,
      Direction.L2ToL1
    )
    const fee = receipts.tx.gasLimit.mul(receipts.tx.gasPrice)

    const postBalances = await getBalances(env)

    expect(postBalances.l1BridgeBalance).to.deep.eq(
      preBalances.l1BridgeBalance.sub(withdrawAmount)
    )
    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.sub(withdrawAmount.add(fee))
    )
    expect(postBalances.l1BobBalance).to.deep.eq(
      preBalances.l1BobBalance.add(withdrawAmount)
    )
  })

  it('deposit, transfer, withdraw', async function () {
    await useDynamicTimeoutForWithdrawals(this, env)

    // 1. deposit
    const amount = utils.parseEther('1')
    await env.waitForXDomainTransaction(
      env.l1Bridge.depositETH(DEFAULT_TEST_GAS_L2, '0xFFFF', {
        value: amount,
        gasLimit: DEFAULT_TEST_GAS_L1,
      }),
      Direction.L1ToL2
    )

    // 2. trnsfer to another address
    const other = Wallet.createRandom().connect(env.l2Wallet.provider)
    const tx = await env.ovmEth.transfer(other.address, amount)
    await tx.wait()

    const l1BalanceBefore = await other
      .connect(env.l1Wallet.provider)
      .getBalance()

    // 3. do withdrawal
    const withdrawnAmount = utils.parseEther('0.95')
    const transaction = await env.l2Bridge
      .connect(other)
      .withdraw(
        predeploys.OVM_ETH,
        withdrawnAmount,
        DEFAULT_TEST_GAS_L1,
        '0xFFFF'
      )
    await transaction.wait()
    await env.relayXDomainMessages(transaction)
    const receipts = await env.waitForXDomainTransaction(
      transaction,
      Direction.L2ToL1
    )

    // check that correct amount was withdrawn and that fee was charged
    const fee = receipts.tx.gasLimit.mul(receipts.tx.gasPrice)
    const l1BalanceAfter = await other
      .connect(env.l1Wallet.provider)
      .getBalance()
    const l2BalanceAfter = await other.getBalance()
    expect(l1BalanceAfter).to.deep.eq(l1BalanceBefore.add(withdrawnAmount))
    expect(l2BalanceAfter).to.deep.eq(amount.sub(withdrawnAmount).sub(fee))
  })

  describe('WETH9 functionality', async () => {
    let initialBalance: BigNumber
    const value = 10

    beforeEach(async () => {
      await fundUser(env.watcher, env.l1Bridge, value, env.l2Wallet.address)
      initialBalance = await env.l2Wallet.provider.getBalance(
        env.l2Wallet.address
      )
    })

    it('successfully deposits', async () => {
      const depositTx = await env.ovmEth.deposit({ value, gasPrice: 0 })
      const receipt = await depositTx.wait()

      expect(
        await env.l2Wallet.provider.getBalance(env.l2Wallet.address)
      ).to.equal(initialBalance)
      expect(receipt.events.length).to.equal(4)

      // The first transfer event is fee payment
      const [, firstTransferEvent, secondTransferEvent, depositEvent] =
        receipt.events

      expect(firstTransferEvent.event).to.equal('Transfer')
      expect(firstTransferEvent.args.from).to.equal(env.l2Wallet.address)
      expect(firstTransferEvent.args.to).to.equal(env.ovmEth.address)
      expect(firstTransferEvent.args.value).to.equal(value)

      expect(secondTransferEvent.event).to.equal('Transfer')
      expect(secondTransferEvent.args.from).to.equal(env.ovmEth.address)
      expect(secondTransferEvent.args.to).to.equal(env.l2Wallet.address)
      expect(secondTransferEvent.args.value).to.equal(value)

      expect(depositEvent.event).to.equal('Deposit')
      expect(depositEvent.args.dst).to.equal(env.l2Wallet.address)
      expect(depositEvent.args.wad).to.equal(value)
    })

    it('successfully deposits on fallback', async () => {
      const fallbackTx = await env.l2Wallet.sendTransaction({
        to: env.ovmEth.address,
        value,
        gasPrice: 0,
      })
      const receipt = await fallbackTx.wait()
      expect(receipt.status).to.equal(1)
      expect(
        await env.l2Wallet.provider.getBalance(env.l2Wallet.address)
      ).to.equal(initialBalance)
    })

    it('successfully withdraws', async () => {
      const withdrawTx = await env.ovmEth.withdraw(value, { gasPrice: 0 })
      const receipt = await withdrawTx.wait()
      expect(
        await env.l2Wallet.provider.getBalance(env.l2Wallet.address)
      ).to.equal(initialBalance)
      expect(receipt.events.length).to.equal(2)

      // The first transfer event is fee payment
      const depositEvent = receipt.events[1]
      expect(depositEvent.event).to.equal('Withdrawal')
      expect(depositEvent.args.src).to.equal(env.l2Wallet.address)
      expect(depositEvent.args.wad).to.equal(value)
    })

    it('reverts on invalid withdraw', async () => {
      await expect(env.ovmEth.withdraw(initialBalance.add(1), { gasPrice: 0 }))
        .to.be.reverted
    })
  })
})
