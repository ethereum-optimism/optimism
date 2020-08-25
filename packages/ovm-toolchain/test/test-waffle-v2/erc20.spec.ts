import { expect } from '../common/setup'

/* External Imports */
import { deployContract } from 'ethereum-waffle-v2'
import { Wallet, Contract } from 'ethers-v4'

/* Internal Imports */
import { waffleV2 } from '../../src/waffle/waffle-v2'

/* Contract Imports */
import * as ERC20 from '../temp/build/waffle/ERC20.json'

const overrides = {
  gasLimit: 10000000,
}

describe('ERC20 smart contract', () => {
  let provider: any
  let wallet1: Wallet
  let wallet2: Wallet
  before(async () => {
    provider = new waffleV2.MockProvider({
      gasLimit: 10000000,
    })
    ;[wallet1, wallet2] = provider.getWallets()
  })

  // parameters to use for our test coin
  const COIN_NAME = 'OVM Test Coin'
  const TICKER = 'OVM'
  const NUM_DECIMALS = 1

  /* Deploy a new ERC20 Token before each test */
  let ERC20Token: Contract
  beforeEach(async () => {
    ERC20Token = await deployContract(
      wallet1,
      ERC20,
      [10000, COIN_NAME, NUM_DECIMALS, TICKER],
      overrides
    )
  })

  it('creation: should create an initial balance of 10000 for the creator', async () => {
    const balance = await ERC20Token.balanceOf(wallet1.address)
    expect(balance.toNumber()).to.equal(10000)
  })

  it('creation: test correct setting of vanity information', async () => {
    const name = await ERC20Token.name()
    expect(name).to.equal(COIN_NAME)

    const decimals = await ERC20Token.decimals()
    expect(decimals).to.equal(NUM_DECIMALS)

    const symbol = await ERC20Token.symbol()
    expect(symbol).to.equal(TICKER)
  })

  it('transfers: should transfer 10000 to walletTo with wallet having 10000', async () => {
    await ERC20Token.transfer(wallet2.address, 10000, overrides)
    const walletToBalance = await ERC20Token.balanceOf(wallet2.address)
    const walletFromBalance = await ERC20Token.balanceOf(wallet1.address)
    expect(walletToBalance.toNumber()).to.equal(10000)
    expect(walletFromBalance.toNumber()).to.equal(0)
  })
})
