/* Imports: External */
import { BigNumber, utils } from 'ethers'
import { serialize } from '@ethersproject/transactions'
import { predeploys, getContractFactory } from '@eth-optimism/contracts'

/* Imports: Internal */
import { expect } from './shared/setup'
import { hardhatTest, gasPriceOracleWallet } from './shared/utils'
import { OptimismEnv } from './shared/env'

const setPrices = async (env: OptimismEnv, value: number | BigNumber) => {
  const gasPrice = await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
    gasPriceOracleWallet
  ).setGasPrice(value)
  await gasPrice.wait()
  const baseFee = await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
    gasPriceOracleWallet
  ).setL1BaseFee(value)
  await baseFee.wait()
}

describe('Fee Payment Integration Tests', async () => {
  let env: OptimismEnv
  const other = '0x1234123412341234123412341234123412341234'

  before(async () => {
    env = await OptimismEnv.new()
  })

  hardhatTest(
    `should return eth_gasPrice equal to OVM_GasPriceOracle.gasPrice`,
    async () => {
      const assertGasPrice = async () => {
        const gasPrice = await env.l2Wallet.getGasPrice()
        const oracleGasPrice =
          await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
            gasPriceOracleWallet
          ).gasPrice()
        expect(gasPrice).to.deep.equal(oracleGasPrice)
      }

      await assertGasPrice()
      // update the gas price
      const tx = await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
        gasPriceOracleWallet
      ).setGasPrice(1000)
      await tx.wait()

      await assertGasPrice()
    }
  )

  hardhatTest('Paying a nonzero but acceptable gasPrice fee', async () => {
    await setPrices(env, 1000)

    const amount = utils.parseEther('0.0000001')
    const balanceBefore = await env.l2Wallet.getBalance()
    const feeVaultBalanceBefore = await env.l2Wallet.provider.getBalance(
      env.messenger.contracts.l2.OVM_SequencerFeeVault.address
    )
    expect(balanceBefore.gt(amount))

    const unsigned = await env.l2Wallet.populateTransaction({
      to: other,
      value: amount,
      gasLimit: 500000,
    })

    const raw = serialize({
      nonce: parseInt(unsigned.nonce.toString(10), 10),
      value: unsigned.value,
      gasPrice: unsigned.gasPrice,
      gasLimit: unsigned.gasLimit,
      to: unsigned.to,
      data: unsigned.data,
    })

    const l1Fee = await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
      gasPriceOracleWallet
    ).getL1Fee(raw)

    const tx = await env.l2Wallet.sendTransaction(unsigned)
    const receipt = await tx.wait()
    expect(receipt.status).to.eq(1)

    const balanceAfter = await env.l2Wallet.getBalance()
    const feeVaultBalanceAfter = await env.l2Wallet.provider.getBalance(
      env.messenger.contracts.l2.OVM_SequencerFeeVault.address
    )

    const l2Fee = receipt.gasUsed.mul(tx.gasPrice)

    const expectedFeePaid = l1Fee.add(l2Fee)

    expect(balanceBefore.sub(balanceAfter)).to.deep.equal(
      expectedFeePaid.add(amount)
    )

    // Make sure the fee was transferred to the vault.
    expect(feeVaultBalanceAfter.sub(feeVaultBalanceBefore)).to.deep.equal(
      expectedFeePaid
    )

    await setPrices(env, 1)
  })

  hardhatTest('should compute correct fee', async () => {
    await setPrices(env, 1000)

    const preBalance = await env.l2Wallet.getBalance()

    const OVM_GasPriceOracle = getContractFactory('OVM_GasPriceOracle')
      .attach(predeploys.OVM_GasPriceOracle)
      .connect(env.l2Wallet)

    const WETH = getContractFactory('OVM_ETH')
      .attach(predeploys.OVM_ETH)
      .connect(env.l2Wallet)

    const feeVaultBefore = await WETH.balanceOf(
      predeploys.OVM_SequencerFeeVault
    )

    const unsigned = await env.l2Wallet.populateTransaction({
      to: env.l2Wallet.address,
      value: 0,
    })

    const raw = serialize({
      nonce: parseInt(unsigned.nonce.toString(10), 10),
      value: unsigned.value,
      gasPrice: unsigned.gasPrice,
      gasLimit: unsigned.gasLimit,
      to: unsigned.to,
      data: unsigned.data,
    })

    const l1Fee = await OVM_GasPriceOracle.connect(
      gasPriceOracleWallet
    ).getL1Fee(raw)

    const tx = await env.l2Wallet.sendTransaction(unsigned)
    const receipt = await tx.wait()
    const l2Fee = receipt.gasUsed.mul(tx.gasPrice)
    const postBalance = await env.l2Wallet.getBalance()
    const feeVaultAfter = await WETH.balanceOf(predeploys.OVM_SequencerFeeVault)
    const fee = l1Fee.add(l2Fee)
    const balanceDiff = preBalance.sub(postBalance)
    const feeReceived = feeVaultAfter.sub(feeVaultBefore)
    expect(balanceDiff).to.deep.equal(fee)
    // There is no inflation
    expect(feeReceived).to.deep.equal(balanceDiff)

    await setPrices(env, 1)
  })

  it('should not be able to withdraw fees before the minimum is met', async () => {
    await expect(env.messenger.contracts.l2.OVM_SequencerFeeVault.withdraw()).to
      .be.rejected
  })

  hardhatTest(
    'should be able to withdraw fees back to L1 once the minimum is met',
    async () => {
      const l1FeeWallet =
        await env.messenger.contracts.l2.OVM_SequencerFeeVault.l1FeeWallet()
      const balanceBefore = await env.l1Wallet.provider.getBalance(l1FeeWallet)
      const withdrawalAmount =
        await env.messenger.contracts.l2.OVM_SequencerFeeVault.MIN_WITHDRAWAL_AMOUNT()

      // Transfer the minimum required to withdraw.
      const tx = await env.l2Wallet.sendTransaction({
        to: env.messenger.contracts.l2.OVM_SequencerFeeVault.address,
        value: withdrawalAmount,
        gasLimit: 500000,
      })
      await tx.wait()

      const vaultBalance = await env.messenger.contracts.l2.OVM_ETH.balanceOf(
        env.messenger.contracts.l2.OVM_SequencerFeeVault.address
      )

      const withdrawTx =
        await env.messenger.contracts.l2.OVM_SequencerFeeVault.withdraw()

      // Wait for the withdrawal to be relayed to L1.
      await withdrawTx.wait()
      await env.relayXDomainMessages(withdrawTx)
      await env.waitForXDomainTransaction(withdrawTx)

      // Balance difference should be equal to old L2 balance.
      const balanceAfter = await env.l1Wallet.provider.getBalance(l1FeeWallet)
      expect(balanceAfter.sub(balanceBefore)).to.deep.equal(
        BigNumber.from(vaultBalance)
      )
    }
  )
})
