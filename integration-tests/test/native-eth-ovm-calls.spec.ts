import { BigNumber, Contract, ContractFactory, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import chai, { expect } from 'chai'
import { GWEI, fundUser } from './shared/utils'
import { OptimismEnv } from './shared/env'
import { solidity } from 'ethereum-waffle'

chai.use(solidity)

describe('OVM calls with native ETH value', async () => {
  const initialBalance0 = 42000

  let env: OptimismEnv
  let wallet: Wallet
  let other: Wallet
  let Factory__ValueCalls: ContractFactory
  let ValueCalls0: Contract
  let ValueCalls1: Contract

  const checkBalances = async (expectedBalances: number[]) => {
    const balance0 = await wallet.provider.getBalance(ValueCalls0.address)
    expect(balance0).to.deep.eq(BigNumber.from(expectedBalances[0]))
    const balance1 = await wallet.provider.getBalance(ValueCalls1.address)
    expect(balance1).to.deep.eq(BigNumber.from(expectedBalances[1]))
  }

  before(async () => {
    env = await OptimismEnv.new()
    wallet = env.l2Wallet
    other = Wallet.createRandom().connect(ethers.provider)
    Factory__ValueCalls = await ethers.getContractFactory('ValueCalls', wallet)
  })

  beforeEach(async () => {
    ValueCalls0 = await Factory__ValueCalls.deploy()
    ValueCalls1 = await Factory__ValueCalls.deploy()
    await fundUser(
      env.watcher,
      env.gateway,
      initialBalance0,
      ValueCalls0.address
    )
    // These tests ass assume ValueCalls0 starts with a balance, but ValueCalls1 does not.
    await checkBalances([initialBalance0, 0])
  })

  it('should set the total supply', async () => {
    console.log('we did it reddit?')
  })

})
