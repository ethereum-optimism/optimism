import chai, { expect } from 'chai'
import chaiAsPromised from 'chai-as-promised'
chai.use(chaiAsPromised)

/* Imports: External */
import { ethers } from 'hardhat'
import {
  BigNumber,
  Contract,
  utils,
  Wallet,
  constants,
  ContractFactory,
} from 'ethers'
import {
  TxGasLimit,
  TxGasPrice,
  LibEIP155TxStruct,
} from '@eth-optimism/core-utils'
import {
  predeploys,
  getContractInterface,
  getContractFactory,
} from '@eth-optimism/contracts'

/* Imports: Internal */
import { OptimismEnv } from './shared/env'
import { Direction } from './shared/watcher-utils'
import { DEFAULT_TRANSACTION, expectApprox } from './shared/utils'

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

  it('should use the correctly estimated intrinsic gas for transactions of varying lengths', async () => {
    // Make test path independent by sending a random transaction, causing the ovmCREATEEOA to occur
    await env.ovmEth
      .connect(wallet)
      .transfer('0x1234123412341234123412341234123412341234', 0) // TODO: make it so that we can use zero address here

    // Get a test library for knowing the value returned by IntrinsicGas.ecdsaContractAccount(...)
    const Factory__TestLib_IntrinsicGas: ContractFactory = getContractFactory(
      'TestLib_IntrinsicGas',
      wallet,
      true
    )
    const TestLib_IntrinsicGas: Contract = await Factory__TestLib_IntrinsicGas.deploy()
    await TestLib_IntrinsicGas.deployTransaction.wait()

    // Get a gas measurer helper contract
    const Factory__GasMeasurer: ContractFactory = await ethers.getContractFactory(
      'GasMeasurer',
      wallet
    )
    const GasMeasurer: Contract = await Factory__GasMeasurer.deploy()
    await GasMeasurer.deployTransaction.wait()

    // Get a modified proxyEOA so that we can subtract out that gas cost.
    // Modification is to set the implementation to a random account so that the delegatecall is empty.
    const Factory__SetStorageAndDeployCode = await ethers.getContractFactory(
      'SetStorageAndDeployCode',
      wallet
    )
    const proxyEOACode = await wallet.provider.getCode(wallet.address)
    const mockProxyEOA = await Factory__SetStorageAndDeployCode.deploy(
      '0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc', // proxyEOA IMPLEMENTATION_KEY
      ethers.constants.MaxUint256, // Nonexistent implementation to minimize cost
      proxyEOACode
    )
    await mockProxyEOA.deployTransaction.wait()
    // sanity check we did this right, we should now have a proxyEOA with implementation address at MaxUint256
    expect(await wallet.provider.getCode(mockProxyEOA.address)).to.deep.eq(
      proxyEOACode
    )

    const iOVM_ECDSAContractAccount = getContractInterface(
      'OVM_ECDSAContractAccount',
      true
    )

    const BASE_TRANSACTION = {
      to: '0x' + '1234'.repeat(10), // Using a code-less address will consume minimal possible gas.
      gasLimit: 33600000000001,
      gasPrice: 0,
      data: '0x',
      value: 0,
      chainId: await wallet.getChainId(),
    }

    for (const dataSize of [
      10,
      100,
      1000,
      10000,
      25000,
      50000,
      75000,
      100000,
      127000,
    ]) {
      // Manually generate the calldata for the transaction which would normally be
      // generated by the OVM_SequencerEntrypoint, so that we can call via GasMeasurer instead.
      const transaction = {
        ...BASE_TRANSACTION,
        nonce: await wallet.getTransactionCount(),
        data: '0x' + Buffer.alloc(dataSize, 0xff).toString('hex'),
      }
      const executableTransaction = LibEIP155TxStruct(
        await wallet.signTransaction(transaction)
      )
      const calldataToContractAccount = iOVM_ECDSAContractAccount.encodeFunctionData(
        'execute',
        [executableTransaction]
      )

      // Measure the gas cost of the call, subtracting out cost of the proxy.
      const gasCostIncludingProxy = await GasMeasurer.callStatic.measureGasCostOfCall(
        wallet.address,
        calldataToContractAccount,
        { gasLimit: 20_000_000 }
      )
      const gasCostOfMockProxy = await GasMeasurer.callStatic.measureGasCostOfCall(
        mockProxyEOA.address,
        calldataToContractAccount,
        { gasLimit: 20_000_000 }
      )
      const actualCost = gasCostIncludingProxy.sub(gasCostOfMockProxy)

      // Compare this to the library's estimated intrinsic cost.
      const estimatedIntrinsicGas = await TestLib_IntrinsicGas.ecdsaContractAccount(
        dataSize,
        { gasLimit: 10_000_000 }
      )
      expect(actualCost).instanceof(BigNumber)
      expectApprox(actualCost, estimatedIntrinsicGas, 1, 1)
    }
  })
})
