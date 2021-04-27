import { expect } from 'chai'
import { Wallet, utils, BigNumber } from 'ethers'
import { Direction } from './shared/watcher-utils'

import { OptimismEnv } from './shared/env'

describe('Native ETH Integration Tests', async () => {

  let env: OptimismEnv
  //let l1Bob: Wallet
  //let l2Bob: Wallet

  const getBalances = async (_env: OptimismEnv) => {

    const l1UserBalance = await _env.bobl1Wallet.getBalance()
    // console.log("sequencerl1Balance:",l1UserBalance.toString())

    const l1GatewayBalance = await _env.bobl1Wallet.provider.getBalance(_env.L1ETHGateway.address)
    // console.log("l1GatewayBalance:",l1GatewayBalance.toString())

    const alicel1Balance = await _env.alicel1Wallet.getBalance()
    // console.log("alicel1Balance:",alicel1Balance.toString())

    const alicel2Balance = await _env.alicel2Wallet.getBalance()
    // console.log("alicel2Balance:",alicel2Balance.toString())

    // const l1UserBalance = await _env.bobl1Wallet.getBalance()
    // const l2UserBalance = await _env.l2Wallet.getBalance()
    // const l1BobBalance = await l1Bob.getBalance()
    // const l2BobBalance = await l2Bob.getBalance()
    // const sequencerBalance = await _env.ovmEth.balanceOf(PROXY_SEQUENCER_ENTRYPOINT_ADDRESS)
    // const l1GatewayBalance = await _env.bobl1Wallet.provider.getBalance(_env.gateway.address)

    return {
      l1UserBalance,      
      alicel1Balance,
      alicel2Balance,
      //l2UserBalance,
      //l1BobBalance,
      //l2BobBalance,
      l1GatewayBalance,
      //sequencerBalance,
    }
  }

  before(async () => {
    env = await OptimismEnv.new()
  })

  it('deposit', async () => {

    const depositAmount = BigNumber.from(15)
    const preBalances = await getBalances(env)
    
    console.log(" Depositing...")

    const { tx, receipt } = await env.waitForXDomainTransaction(
      env.L1ETHGateway.deposit({ value: depositAmount }),
      Direction.L1ToL2
    )

    const l1FeePaid = receipt.gasUsed.mul(tx.gasPrice)
    const postBalances = await getBalances(env)

    expect(postBalances.l1GatewayBalance).to.deep.eq(
      preBalances.l1GatewayBalance.add(depositAmount)
    )
    expect(postBalances.alicel2Balance).to.deep.eq(
      preBalances.alicel2Balance.add(depositAmount)
    )
    expect(postBalances.alicel1Balance).to.deep.eq(
      preBalances.alicel1Balance.sub(l1FeePaid.add(depositAmount))
    )
  })

  it('withdraw', async () => {

    const withdrawAmount = BigNumber.from(10)
    const preBalances = await getBalances(env)
    
    expect(
      preBalances.alicel2Balance.gt(0),
      ' Sorry Alice is broke - cannot withdraw...'
    )
    
    console.log(" Withdrawing...")
    
    const receipts = await env.waitForXDomainTransaction(
      env.L2ETHGateway.withdraw(withdrawAmount),
      Direction.L2ToL1
    )

    const fee = receipts.tx.gasLimit.mul(receipts.tx.gasPrice)

    const postBalances = await getBalances(env)

    console.log({
      l1GatewayBalancePre: preBalances.l1GatewayBalance.toString(),
      l1GatewayBalancePost: postBalances.l1GatewayBalance.toString(),
    })

    console.log({
      alicel2BalancePre: preBalances.alicel2Balance.toString(),
      alicel2BalancePost: postBalances.alicel2Balance.toString(),
      fee: fee.toString(),
    })

    console.log({
      alicel1BalancePre: preBalances.alicel1Balance.toString(),
      alicel1BalancePost: postBalances.alicel1Balance.toString(),
    })

    expect(postBalances.l1GatewayBalance).to.deep.eq(
      preBalances.l1GatewayBalance.sub(withdrawAmount)
    )
    expect(postBalances.alicel2Balance).to.deep.eq(
      preBalances.alicel2Balance.sub(withdrawAmount).sub(fee)
    )
    expect(postBalances.alicel1Balance).to.deep.eq(
      preBalances.alicel1Balance.add(withdrawAmount)
    )
  })
})
