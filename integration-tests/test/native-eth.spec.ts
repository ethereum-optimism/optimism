/* Imports: External */
import { Wallet, utils, BigNumber } from 'ethers'
import { serialize } from '@ethersproject/transactions'
import { predeploys } from '@eth-optimism/contracts'
import { expectApprox } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { expect } from './shared/setup'
import {
  DEFAULT_TEST_GAS_L1,
  DEFAULT_TEST_GAS_L2,
  envConfig,
  withdrawalTest,
  gasPriceOracleWallet,
} from './shared/utils'
import { OptimismEnv } from './shared/env'

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

    const l1BridgeBalance = await _env.l1Wallet.provider.getBalance(
      _env.messenger.contracts.l1.L1StandardBridge.address
    )

    return {
      l1UserBalance,
      l2UserBalance,
      l1BobBalance,
      l2BobBalance,
      l1BridgeBalance,
    }
  }

  before(async () => {
    env = await OptimismEnv.new()
    l1Bob = Wallet.createRandom().connect(env.l1Wallet.provider)
    l2Bob = l1Bob.connect(env.l2Wallet.provider)
  })

  describe('estimateGas', () => {
    it('Should estimate gas for ETH withdraw', async () => {
      const amount = utils.parseEther('0.0000001')
      const gas =
        await env.messenger.contracts.l2.L2StandardBridge.estimateGas.withdraw(
          predeploys.OVM_ETH,
          amount,
          0,
          '0xFFFF'
        )
      // Expect gas to be less than or equal to the target plus 1%
      expectApprox(gas, 6700060, { absoluteUpperDeviation: 1000 })
    })
  })

  it('receive', async () => {
    const depositAmount = 10
    const preBalances = await getBalances(env)
    const { tx, receipt } = await env.waitForXDomainTransaction(
      env.l1Wallet.sendTransaction({
        to: env.messenger.contracts.l1.L1StandardBridge.address,
        value: depositAmount,
        gasLimit: DEFAULT_TEST_GAS_L1,
      })
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
      env.messenger.contracts.l1.L1StandardBridge.depositETH(
        DEFAULT_TEST_GAS_L2,
        '0xFFFF',
        {
          value: depositAmount,
          gasLimit: DEFAULT_TEST_GAS_L1,
        }
      )
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
      env.messenger.contracts.l1.L1StandardBridge.depositETHTo(
        l2Bob.address,
        DEFAULT_TEST_GAS_L2,
        '0xFFFF',
        {
          value: depositAmount,
          gasLimit: DEFAULT_TEST_GAS_L1,
        }
      )
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
      env.messenger.contracts.l1.L1StandardBridge.depositETH(
        ASSUMED_L2_GAS_LIMIT,
        data,
        {
          value: depositAmount,
          gasLimit: 4_000_000,
        }
      )
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
      env.messenger.contracts.l1.L1StandardBridge.depositETH(
        DEFAULT_TEST_GAS_L2,
        data,
        {
          value: depositAmount,
        }
      )
    ).to.be.reverted
  })

  withdrawalTest('withdraw', async () => {
    const withdrawAmount = BigNumber.from(3)
    const preBalances = await getBalances(env)
    expect(
      preBalances.l2UserBalance.gt(0),
      'Cannot run withdrawal test before any deposits...'
    )

    const transaction =
      await env.messenger.contracts.l2.L2StandardBridge.withdraw(
        predeploys.OVM_ETH,
        withdrawAmount,
        DEFAULT_TEST_GAS_L2,
        '0xFFFF'
      )
    await transaction.wait()
    await env.relayXDomainMessages(transaction)
    const receipts = await env.waitForXDomainTransaction(transaction)
    const fee = receipts.tx.gasLimit.mul(receipts.tx.gasPrice)

    const postBalances = await getBalances(env)

    // Approximate because there's a fee related to relaying the L2 => L1 message and it throws off the math.
    expectApprox(
      postBalances.l1BridgeBalance,
      preBalances.l1BridgeBalance.sub(withdrawAmount),
      { percentUpperDeviation: 1 }
    )
    expectApprox(
      postBalances.l2UserBalance,
      preBalances.l2UserBalance.sub(withdrawAmount.add(fee)),
      { percentUpperDeviation: 1 }
    )
    expectApprox(
      postBalances.l1UserBalance,
      preBalances.l1UserBalance.add(withdrawAmount),
      { percentUpperDeviation: 1 }
    )
  })

  withdrawalTest('withdrawTo', async () => {
    const withdrawAmount = BigNumber.from(3)

    const preBalances = await getBalances(env)

    expect(
      preBalances.l2UserBalance.gt(0),
      'Cannot run withdrawal test before any deposits...'
    )

    const transaction =
      await env.messenger.contracts.l2.L2StandardBridge.withdrawTo(
        predeploys.OVM_ETH,
        l1Bob.address,
        withdrawAmount,
        DEFAULT_TEST_GAS_L2,
        '0xFFFF'
      )

    await transaction.wait()
    await env.relayXDomainMessages(transaction)
    const receipts = await env.waitForXDomainTransaction(transaction)

    const l2Fee = receipts.tx.gasPrice.mul(receipts.receipt.gasUsed)

    // Calculate the L1 portion of the fee
    const raw = serialize({
      nonce: transaction.nonce,
      value: transaction.value,
      gasPrice: transaction.gasPrice,
      gasLimit: transaction.gasLimit,
      to: transaction.to,
      data: transaction.data,
    })

    const l1Fee = await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
      gasPriceOracleWallet
    ).getL1Fee(raw)
    const fee = l2Fee.add(l1Fee)

    const postBalances = await getBalances(env)

    expect(postBalances.l1BridgeBalance).to.deep.eq(
      preBalances.l1BridgeBalance.sub(withdrawAmount),
      'L1 Bridge Balance Mismatch'
    )

    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.sub(withdrawAmount.add(fee)),
      'L2 User Balance Mismatch'
    )

    expect(postBalances.l1BobBalance).to.deep.eq(
      preBalances.l1BobBalance.add(withdrawAmount),
      'L1 User Balance Mismatch'
    )
  })

  withdrawalTest(
    'deposit, transfer, withdraw',
    async () => {
      // 1. deposit
      const amount = utils.parseEther('1')
      await env.waitForXDomainTransaction(
        env.messenger.contracts.l1.L1StandardBridge.depositETH(
          DEFAULT_TEST_GAS_L2,
          '0xFFFF',
          {
            value: amount,
            gasLimit: DEFAULT_TEST_GAS_L1,
          }
        )
      )

      // 2. transfer to another address
      const other = Wallet.createRandom().connect(env.l2Wallet.provider)
      const tx = await env.l2Wallet.sendTransaction({
        to: other.address,
        value: amount,
      })
      await tx.wait()

      const l1BalanceBefore = await other
        .connect(env.l1Wallet.provider)
        .getBalance()

      // 3. do withdrawal
      const withdrawnAmount = utils.parseEther('0.95')
      const transaction =
        await env.messenger.contracts.l2.L2StandardBridge.connect(
          other
        ).withdraw(
          predeploys.OVM_ETH,
          withdrawnAmount,
          DEFAULT_TEST_GAS_L1,
          '0xFFFF'
        )
      await transaction.wait()
      await env.relayXDomainMessages(transaction)
      const receipts = await env.waitForXDomainTransaction(transaction)

      // Compute the L1 portion of the fee
      const l1Fee =
        await await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).getL1Fee(
          serialize({
            nonce: transaction.nonce,
            value: transaction.value,
            gasPrice: transaction.gasPrice,
            gasLimit: transaction.gasLimit,
            to: transaction.to,
            data: transaction.data,
          })
        )

      // check that correct amount was withdrawn and that fee was charged
      const l2Fee = receipts.tx.gasPrice.mul(receipts.receipt.gasUsed)

      const fee = l1Fee.add(l2Fee)
      const l1BalanceAfter = await other
        .connect(env.l1Wallet.provider)
        .getBalance()
      const l2BalanceAfter = await other.getBalance()
      expect(l1BalanceAfter).to.deep.eq(l1BalanceBefore.add(withdrawnAmount))
      expect(l2BalanceAfter).to.deep.eq(amount.sub(withdrawnAmount).sub(fee))
    },
    envConfig.MOCHA_TIMEOUT * 3
  )
})
