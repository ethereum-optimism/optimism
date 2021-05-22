import { expect } from 'chai'
import { Wallet, utils, BigNumber } from 'ethers'
import { Direction } from './shared/watcher-utils'

import { PROXY_SEQUENCER_ENTRYPOINT_ADDRESS } from './shared/utils'
import { OptimismEnv } from './shared/env'

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
    const l1GatewayBalance = await _env.l1Wallet.provider.getBalance(
      _env.gateway.address
    )

    return {
      l1UserBalance,
      l2UserBalance,
      l1BobBalance,
      l2BobBalance,
      l1GatewayBalance,
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
      const amount = utils.parseEther('0.5')
      const addr = '0x' + '1234'.repeat(10)
      const gas = await env.ovmEth.estimateGas.transfer(addr, amount)
      expect(gas).to.be.deep.eq(BigNumber.from(21000))
    })

    it('Should estimate gas for ETH withdraw', async () => {
      const amount = utils.parseEther('0.5')
      const gas = await env.ovmEth.estimateGas.withdraw(amount)
      expect(gas).to.be.deep.eq(BigNumber.from(21000))
    })
  })

  it('deposit', async () => {
    const depositAmount = 10
    const preBalances = await getBalances(env)
    const { tx, receipt } = await env.waitForXDomainTransaction(
      env.gateway.deposit({ value: depositAmount }),
      Direction.L1ToL2
    )

    const l1FeePaid = receipt.gasUsed.mul(tx.gasPrice)
    const postBalances = await getBalances(env)

    expect(postBalances.l1GatewayBalance).to.deep.eq(
      preBalances.l1GatewayBalance.add(depositAmount)
    )
    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.add(depositAmount)
    )
    expect(postBalances.l1UserBalance).to.deep.eq(
      preBalances.l1UserBalance.sub(l1FeePaid.add(depositAmount))
    )
  })

  it('depositTo', async () => {
    const depositAmount = 10
    const preBalances = await getBalances(env)
    const depositReceipts = await env.waitForXDomainTransaction(
      env.gateway.depositTo(l2Bob.address, {
        value: depositAmount,
      }),
      Direction.L1ToL2
    )

    const l1FeePaid = depositReceipts.receipt.gasUsed.mul(
      depositReceipts.tx.gasPrice
    )
    const postBalances = await getBalances(env)
    expect(postBalances.l1GatewayBalance).to.deep.eq(
      preBalances.l1GatewayBalance.add(depositAmount)
    )
    expect(postBalances.l2BobBalance).to.deep.eq(
      preBalances.l2BobBalance.add(depositAmount)
    )
    expect(postBalances.l1UserBalance).to.deep.eq(
      preBalances.l1UserBalance.sub(l1FeePaid.add(depositAmount))
    )
  })

  it('withdraw', async () => {
    const withdrawAmount = BigNumber.from(3)
    const preBalances = await getBalances(env)
    expect(
      preBalances.l2UserBalance.gt(0),
      'Cannot run withdrawal test before any deposits...'
    )

    const receipts = await env.waitForXDomainTransaction(
      env.ovmEth.withdraw(withdrawAmount),
      Direction.L2ToL1
    )
    const fee = receipts.tx.gasLimit.mul(receipts.tx.gasPrice)

    const postBalances = await getBalances(env)

    expect(postBalances.l1GatewayBalance).to.deep.eq(
      preBalances.l1GatewayBalance.sub(withdrawAmount)
    )
    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.sub(withdrawAmount.add(fee))
    )
    expect(postBalances.l1UserBalance).to.deep.eq(
      preBalances.l1UserBalance.add(withdrawAmount)
    )
  })

  it('withdrawTo', async () => {
    const withdrawAmount = BigNumber.from(3)

    const preBalances = await getBalances(env)

    expect(
      preBalances.l2UserBalance.gt(0),
      'Cannot run withdrawal test before any deposits...'
    )

    const receipts = await env.waitForXDomainTransaction(
      env.ovmEth.withdrawTo(l1Bob.address, withdrawAmount),
      Direction.L2ToL1
    )
    const fee = receipts.tx.gasLimit.mul(receipts.tx.gasPrice)

    const postBalances = await getBalances(env)

    expect(postBalances.l1GatewayBalance).to.deep.eq(
      preBalances.l1GatewayBalance.sub(withdrawAmount)
    )
    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.sub(withdrawAmount.add(fee))
    )
    expect(postBalances.l1BobBalance).to.deep.eq(
      preBalances.l1BobBalance.add(withdrawAmount)
    )
  })

  it('deposit, transfer, withdraw', async () => {
    // 1. deposit
    const amount = utils.parseEther('1')
    await env.waitForXDomainTransaction(
      env.gateway.deposit({
        value: amount,
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
    const receipts = await env.waitForXDomainTransaction(
      env.ovmEth.connect(other).withdraw(withdrawnAmount),
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
})
