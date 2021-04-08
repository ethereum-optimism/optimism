import chai, { expect } from 'chai'
import chaiAsPromised from 'chai-as-promised'
chai.use(chaiAsPromised)
import { BigNumber, Contract, utils } from 'ethers'

import {
  l1Provider,
  l2Provider,
  l1Wallet,
  l2Wallet,
  getGateway,
  getAddressManager,
  getOvmEth,
  fundUser,
} from './shared/utils'
import { initWatcher } from './shared/watcher-utils'

describe('Fee Payment Integration Tests', async () => {
  let OVM_L1ETHGateway: Contract
  let OVM_ETH: Contract
  let AddressManager: Contract
  const other = '0x1234123412341234123412341234123412341234'
  const amount = utils.parseEther('1')

  before(async () => {
    AddressManager = getAddressManager(l1Wallet)
    OVM_L1ETHGateway = await getGateway(l1Wallet, AddressManager)
    OVM_ETH = getOvmEth(l2Wallet)
    const watcher = await initWatcher(l1Provider, l2Provider, AddressManager)
    await fundUser(watcher, OVM_L1ETHGateway, amount)
  })

  it('Paying a nonzero but acceptable gasPrice fee', async () => {
    // manually set the gas price because otherwise it's returned as 0
    const gasPrice = BigNumber.from(1_000_000)
    const amt = amount.div(2)

    const balanceBefore = await l2Wallet.getBalance()
    const tx = await OVM_ETH.transfer(other, amt, { gasPrice })
    await tx.wait()
    const balanceAfter = await l2Wallet.getBalance()
    expect(balanceBefore.sub(balanceAfter)).to.be.deep.eq(
      gasPrice.mul(tx.gasLimit).add(amt)
    )
  })

  it('sequencer rejects transaction with a non-multiple-of-1M gasPrice', async () => {
    const gasPrice = BigNumber.from(1_000_000 - 1)
    await expect(
      OVM_ETH.transfer(other, 0, { gasPrice })
    ).to.be.eventually.rejectedWith(
      'Gas price must be a multiple of 1,000,000 wei'
    )
  })
})
