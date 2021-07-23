import chai, { expect } from 'chai'
import chaiAsPromised from 'chai-as-promised'
chai.use(chaiAsPromised)

/* Imports: External */
import { ethers, BigNumber, Contract, utils } from 'ethers'
import { TxGasLimit, TxGasPrice } from '@eth-optimism/core-utils'
import { predeploys, getContractInterface } from '@eth-optimism/contracts'

/* Imports: Internal */
import { IS_LIVE_NETWORK } from './shared/utils'
import { OptimismEnv } from './shared/env'
import { Direction } from './shared/watcher-utils'

describe('Fee Payment Integration Tests', async () => {
  const other = '0x1234123412341234123412341234123412341234'

  let env: OptimismEnv
  before(async () => {
    env = await OptimismEnv.new()
  })

  let ovmSequencerFeeVault: Contract
  before(async () => {
    ovmSequencerFeeVault = new Contract(
      predeploys.OVM_SequencerFeeVault,
      getContractInterface('OVM_SequencerFeeVault'),
      env.l2Wallet
    )
  })

  it(`Should return a gasPrice of ${TxGasPrice.toString()} wei`, async () => {
    const gasPrice = await env.l2Wallet.getGasPrice()
    expect(gasPrice).to.deep.eq(TxGasPrice)
  })

  it('Should estimateGas with recoverable L2 gasLimit', async () => {
    const gas = await env.ovmEth.estimateGas.transfer(
      other,
      utils.parseEther('0.0000001')
    )
    const tx = await env.ovmEth.populateTransaction.transfer(
      other,
      utils.parseEther('0.0000001')
    )
    const executionGas = await (env.ovmEth.provider as any).send(
      'eth_estimateExecutionGas',
      [tx, true]
    )
    const decoded = TxGasLimit.decode(gas)
    expect(BigNumber.from(executionGas)).deep.eq(decoded)
  })

  it('Paying a nonzero but acceptable gasPrice fee', async () => {
    const amount = utils.parseEther('0.0000001')
    const balanceBefore = await env.l2Wallet.getBalance()
    const feeVaultBalanceBefore = await env.l2Wallet.provider.getBalance(
      ovmSequencerFeeVault.address
    )
    expect(balanceBefore.gt(amount))

    const tx = await env.ovmEth.transfer(other, amount)
    const receipt = await tx.wait()
    expect(receipt.status).to.eq(1)

    const balanceAfter = await env.l2Wallet.getBalance()
    const feeVaultBalanceAfter = await env.l2Wallet.provider.getBalance(
      ovmSequencerFeeVault.address
    )
    const expectedFeePaid = tx.gasPrice.mul(tx.gasLimit)

    // The fee paid MUST be the receipt.gasUsed, and not the tx.gasLimit
    // https://github.com/ethereum-optimism/optimism/blob/0de7a2f9c96a7c4860658822231b2d6da0fefb1d/packages/contracts/contracts/optimistic-ethereum/OVM/accounts/OVM_ECDSAContractAccount.sol#L103
    expect(balanceBefore.sub(balanceAfter)).to.deep.equal(
      expectedFeePaid.add(amount)
    )

    // Make sure the fee was transferred to the vault.
    expect(feeVaultBalanceAfter.sub(feeVaultBalanceBefore)).to.deep.equal(
      expectedFeePaid
    )
  })

  it('should not be able to withdraw fees before the minimum is met', async () => {
    await expect(ovmSequencerFeeVault.withdraw()).to.be.rejected
  })

  it('should be able to withdraw fees back to L1 once the minimum is met', async function () {
    const l1FeeWallet = await ovmSequencerFeeVault.l1FeeWallet()
    const balanceBefore = await env.l1Wallet.provider.getBalance(l1FeeWallet)
    const withdrawalAmount = await ovmSequencerFeeVault.MIN_WITHDRAWAL_AMOUNT()

    const l2WalletBalance = await env.l2Wallet.getBalance()
    if (IS_LIVE_NETWORK && l2WalletBalance.lt(withdrawalAmount)) {
      console.log(
        `NOTICE: must have at least ${ethers.utils.formatEther(
          withdrawalAmount
        )} ETH on L2 to execute this test, skipping`
      )
      this.skip()
    }

    // Transfer the minimum required to withdraw.
    await env.ovmEth.transfer(ovmSequencerFeeVault.address, withdrawalAmount)

    const vaultBalance = await env.ovmEth.balanceOf(
      ovmSequencerFeeVault.address
    )

    // Submit the withdrawal.
    const withdrawTx = await ovmSequencerFeeVault.withdraw({
      gasPrice: 0, // Need a gasprice of 0 or the balances will include the fee paid during this tx.
    })

    // Wait for the withdrawal to be relayed to L1.
    await env.waitForXDomainTransaction(withdrawTx, Direction.L2ToL1)

    // Balance difference should be equal to old L2 balance.
    const balanceAfter = await env.l1Wallet.provider.getBalance(l1FeeWallet)
    expect(balanceAfter.sub(balanceBefore)).to.deep.equal(
      BigNumber.from(vaultBalance)
    )
  })
})
