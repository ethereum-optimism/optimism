import { expect } from '../common/setup'

/* External Imports */
// tslint:disable-next-line
const { ethers } = require('@nomiclabs/buidler')
import { Contract, Signer } from 'ethers-v5'

describe('ERC20 smart contract', () => {
  let wallet1: Signer
  let wallet2: Signer
  before(async () => {
    ;[wallet1, wallet2] = await ethers.getSigners()
  })

  // parameters to use for our test coin
  const COIN_NAME = 'OVM Test Coin'
  const TICKER = 'OVM'
  const NUM_DECIMALS = 1

  /* Deploy a new ERC20 Token before each test */
  let ERC20Token: Contract
  beforeEach(async () => {
    const ERC20TokenFactory = await ethers.getContractFactory('ERC20')
    ERC20Token = await ERC20TokenFactory.deploy(
      10000,
      COIN_NAME,
      NUM_DECIMALS,
      TICKER
    )
  })

  it('creation: should create an initial balance of 10000 for the creator', async () => {
    const balance = await ERC20Token.balanceOf(await wallet1.getAddress())
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
    await ERC20Token.transfer(await wallet2.getAddress(), 10000)
    const walletToBalance = await ERC20Token.balanceOf(
      await wallet2.getAddress()
    )
    const walletFromBalance = await ERC20Token.balanceOf(
      await wallet1.getAddress()
    )
    expect(walletToBalance.toNumber()).to.equal(10000)
    expect(walletFromBalance.toNumber()).to.equal(0)
  })
})
