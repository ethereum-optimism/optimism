import chai, { expect } from 'chai'
import chaiAsPromised from 'chai-as-promised'
chai.use(chaiAsPromised)
import { BigNumber, utils } from 'ethers'
import { OptimismEnv } from './shared/env'
import { TxGasLimit, TxGasPrice } from '@eth-optimism/core-utils'

describe('Fee Payment Integration Tests', async () => {
  let env: OptimismEnv
  const other = '0x1234123412341234123412341234123412341234'

  before(async () => {
    env = await OptimismEnv.new()
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
    expect(balanceBefore.gt(amount))

    const tx = await env.ovmEth.transfer(other, amount)
    const receipt = await tx.wait()
    expect(receipt.status).to.eq(1)

    const balanceAfter = await env.l2Wallet.getBalance()
    // The fee paid MUST be the receipt.gasUsed, and not the tx.gasLimit
    // https://github.com/ethereum-optimism/optimism/blob/0de7a2f9c96a7c4860658822231b2d6da0fefb1d/packages/contracts/contracts/optimistic-ethereum/OVM/accounts/OVM_ECDSAContractAccount.sol#L103
    expect(balanceBefore.sub(balanceAfter)).to.be.deep.eq(
      tx.gasPrice.mul(tx.gasLimit).add(amount)
    )
  })
})
