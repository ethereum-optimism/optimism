import { expect } from 'chai'
import { BigNumber, Contract, Wallet, constants, providers } from 'ethers'
import {
  getContractInterface,
  getContractFactory,
} from '@eth-optimism/contracts'
import { Watcher } from '@eth-optimism/core-utils'

import {
  Direction,
  initWatcher,
  waitForXDomainTransaction,
} from './shared/watcher-utils'

import {
  l1Provider,
  l2Provider,
  l1Wallet,
  l2Wallet,
  getGateway,
  getAddressManager,
  getOvmEth,
  PROXY_SEQUENCER_ENTRYPOINT_ADDRESS,
} from './shared/utils'

describe('Native ETH Integration Tests', async () => {
  let OVM_L1ETHGateway: Contract
  let OVM_ETH: Contract

  let AddressManager: Contract
  let watcher: Watcher

  const BOB_PRIV_KEY =
    '0x1234123412341234123412341234123412341234123412341234123412341234'
  const l1bob = new Wallet(BOB_PRIV_KEY, l1Provider)
  const l2bob = new Wallet(BOB_PRIV_KEY, l2Provider)

  const getBalances = async () => {
    const l1UserBalance = await l1Wallet.getBalance()
    const l2UserBalance = await l2Wallet.getBalance()

    const l1BobBalance = await l1bob.getBalance()
    const l2BobBalance = await l2bob.getBalance()

    const sequencerBalance = await OVM_ETH.balanceOf(
      PROXY_SEQUENCER_ENTRYPOINT_ADDRESS
    )
    const l1GatewayBalance = await l1Provider.getBalance(
      OVM_L1ETHGateway.address
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
    AddressManager = getAddressManager(l1Wallet)
    OVM_L1ETHGateway = await getGateway(l1Wallet, AddressManager)
    OVM_ETH = getOvmEth(l2Wallet)
    watcher = await initWatcher(l1Provider, l2Provider, AddressManager)
  })

  it('deposit', async () => {
    const depositAmount = 10
    const preBalances = await getBalances()
    const { tx, receipt } = await waitForXDomainTransaction(
      watcher,
      OVM_L1ETHGateway.deposit({
        value: depositAmount,
      }),
      Direction.L1ToL2
    )

    const l1FeePaid = receipt.gasUsed.mul(tx.gasPrice)
    const postBalances = await getBalances()

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
    const preBalances = await getBalances()
    const depositReceipts = await waitForXDomainTransaction(
      watcher,
      OVM_L1ETHGateway.depositTo(l2bob.address, {
        value: depositAmount,
      }),
      Direction.L1ToL2
    )

    const l1FeePaid = depositReceipts.receipt.gasUsed.mul(
      depositReceipts.tx.gasPrice
    )
    const postBalances = await getBalances()
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
    const withdrawAmount = 3
    const preBalances = await getBalances()
    expect(
      preBalances.l2UserBalance.gt(0),
      'Cannot run withdrawal test before any deposits...'
    )

    await waitForXDomainTransaction(
      watcher,
      OVM_ETH.withdraw(withdrawAmount),
      Direction.L2ToL1
    )

    const postBalances = await getBalances()

    expect(postBalances.l1GatewayBalance).to.deep.eq(
      preBalances.l1GatewayBalance.sub(withdrawAmount)
    )
    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.sub(withdrawAmount)
    )
    expect(postBalances.l1UserBalance).to.deep.eq(
      preBalances.l1UserBalance.add(withdrawAmount)
    )
  })

  it('withdrawTo', async () => {
    const withdrawAmount = 3

    const preBalances = await getBalances()

    expect(
      preBalances.l2UserBalance.gt(0),
      'Cannot run withdrawal test before any deposits...'
    )

    await waitForXDomainTransaction(
      watcher,
      OVM_ETH.withdrawTo(l1bob.address, withdrawAmount),
      Direction.L2ToL1
    )

    const postBalances = await getBalances()

    expect(postBalances.l1GatewayBalance).to.deep.eq(
      preBalances.l1GatewayBalance.sub(withdrawAmount)
    )
    expect(postBalances.l2UserBalance).to.deep.eq(
      preBalances.l2UserBalance.sub(withdrawAmount)
    )
    expect(postBalances.l1BobBalance).to.deep.eq(
      preBalances.l1BobBalance.add(withdrawAmount)
    )
  })

  it('deposit, transfer, withdraw', async () => {
    const roundTripAmount = 3
    const preBalances = await getBalances()

    await waitForXDomainTransaction(
      watcher,
      OVM_L1ETHGateway.deposit({
        value: roundTripAmount,
      }),
      Direction.L1ToL2
    )

    await OVM_ETH.transfer(l2bob.address, roundTripAmount)

    await waitForXDomainTransaction(
      watcher,
      OVM_ETH.connect(l2bob).withdraw(roundTripAmount),
      Direction.L2ToL1
    )

    const postBalances = await getBalances()

    expect(postBalances.l1BobBalance).to.deep.eq(
      preBalances.l1BobBalance.add(roundTripAmount)
    )
  })
})
