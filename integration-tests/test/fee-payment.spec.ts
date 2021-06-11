import chai, { expect } from 'chai'
import chaiAsPromised from 'chai-as-promised'
chai.use(chaiAsPromised)

/* Imports: External */
import { ethers } from 'hardhat'
import { BigNumber, Contract, utils, Wallet, constants, ContractFactory } from 'ethers'
import { TxGasLimit, TxGasPrice, LibEIP155TxStruct } from '@eth-optimism/core-utils'
import { predeploys, getContractInterface } from '@eth-optimism/contracts'

/* Imports: Internal */
import { OptimismEnv } from './shared/env'
import { Direction } from './shared/watcher-utils'

// TODO: import me from core-utils via kevin's incoming PR
const DEFAULT_EIP155_TX = {
  to: `0x${'12'.repeat(20)}`,
  nonce: 100,
  gasLimit: 1000000,
  gasPrice: 100000000,
  data: `0x${'99'.repeat(10)}`,
  chainId: 420,
}

describe('Fee Payment Integration Tests', async () => {
  const other = '0x1234123412341234123412341234123412341234'

  let env: OptimismEnv
  let wallet: Wallet
  before(async () => {
    env = await OptimismEnv.new()
    wallet = env.l2Wallet
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
      utils.parseEther('0.5')
    )
    const tx = await env.ovmEth.populateTransaction.transfer(
      other,
      utils.parseEther('0.5')
    )
    const executionGas = await (env.ovmEth
      .provider as any).send('eth_estimateExecutionGas', [tx, true])
    const decoded = TxGasLimit.decode(gas)
    expect(BigNumber.from(executionGas)).deep.eq(decoded)
  })

  it('Paying a nonzero but acceptable gasPrice fee', async () => {
    const amount = utils.parseEther('0.5')
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

  it('should be able to withdraw fees back to L1 once the minimum is met', async () => {
    const l1FeeWallet = await ovmSequencerFeeVault.l1FeeWallet()
    const balanceBefore = await env.l1Wallet.provider.getBalance(l1FeeWallet)

    // Transfer the minimum required to withdraw.
    await env.ovmEth.transfer(
      ovmSequencerFeeVault.address,
      await ovmSequencerFeeVault.MIN_WITHDRAWAL_AMOUNT()
    )

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

  it.only('should use gas not exceeding EXECUTE_INTRINSIC_GAS for a 0-gas transaction', async () => {
    // Make test path independent by sending a random transaction, causing the ovmCREATEEOA to occur
    const res = await env.ovmEth.connect(wallet).transfer(constants.AddressZero, 0)

    const iOVM_ECDSAContractAccount = getContractInterface('OVM_ECDSAContractAccount', true)
    const iOVM_ProxyEOA = getContractInterface('OVM_ProxyEOA', true)

    const OVM_ECDSAContractAccount = new Contract(
      await wallet.getAddress(),
      iOVM_ECDSAContractAccount,
      wallet
    )

    const OVM_ProxyEOA = new Contract(
      await wallet.getAddress(),
      iOVM_ProxyEOA,
      wallet
    )

    const EXECUTE_INTRINSIC_GAS = await OVM_ECDSAContractAccount.callStatic.EXECUTE_INTRINSIC_GAS()

    const Factory__GasMeasurer: ContractFactory = await ethers.getContractFactory('GasMeasurer', wallet)
    const GasMeasurer: Contract = await Factory__GasMeasurer.deploy()
    await GasMeasurer.deployTransaction.wait()

    const transaction = {
      ...DEFAULT_EIP155_TX,
      to: constants.AddressZero, // this will consume the minimal gas possible
      data: '0x' + Buffer.alloc(127000).toString('hex'),
    }
    const executableTransaction = LibEIP155TxStruct(await wallet.signTransaction(transaction))
    const calldataToContractAccount = iOVM_ECDSAContractAccount.encodeFunctionData(
      'execute',
      [executableTransaction]
    )

    const gasCost = await GasMeasurer.callStatic.measureGasCostOfCall(
      wallet.address,
      calldataToContractAccount,
      {
        gasLimit: 4_000_000
      }
    )

    const proxy = await GasMeasurer.callStatic.measureGasCostOfCall(
      wallet.address,
      iOVM_ProxyEOA.encodeFunctionData('getImplementation'),
      []
    )

    console.log(EXECUTE_INTRINSIC_GAS.toString())
    console.log(gasCost.toString())
    //expect(gasCost).instanceof(BigNumber)
    //expect(gasCost.sub(proxy).lte(EXECUTE_INTRINSIC_GAS)).to.be.true
  })
})
