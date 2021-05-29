import { Contract, ContractFactory, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import { TxGasLimit, TxGasPrice } from '@eth-optimism/core-utils'
import chai, { expect } from 'chai'
import { GWEI } from './shared/utils'
import { OptimismEnv } from './shared/env'
import { solidity } from 'ethereum-waffle'

chai.use(solidity)

describe('Basic ERC20 interactions', async () => {
  const initialAmount = 1000
  const tokenName = 'OVM Test'
  const tokenDecimals = 8
  const TokenSymbol = 'OVM'

  let wallet: Wallet
  let other: Wallet
  let Factory__ERC20: ContractFactory
  let ERC20: Contract

  before(async () => {
    const env = await OptimismEnv.new()
    wallet = env.l2Wallet
    other = Wallet.createRandom().connect(ethers.provider)
    Factory__ERC20 = await ethers.getContractFactory('ERC20', wallet)
  })

  beforeEach(async () => {
    ERC20 = await Factory__ERC20.deploy(
      initialAmount,
      tokenName,
      tokenDecimals,
      TokenSymbol
    )
  })

  it('should set the total supply', async () => {
    const totalSupply = await ERC20.totalSupply()
    expect(totalSupply.toNumber()).to.equal(initialAmount)
  })

  it('should get the token name', async () => {
    const name = await ERC20.name()
    expect(name).to.equal(tokenName)
  })

  it('should get the token decimals', async () => {
    const decimals = await ERC20.decimals()
    expect(decimals).to.equal(tokenDecimals)
  })

  it('should get the token symbol', async () => {
    const symbol = await ERC20.symbol()
    expect(symbol).to.equal(TokenSymbol)
  })

  it('should assign initial balance', async () => {
    const balance = await ERC20.balanceOf(wallet.address)
    expect(balance.toNumber()).to.equal(initialAmount)
  })

  it('should transfer amount to destination account', async () => {
    const transfer = await ERC20.transfer(other.address, 100)
    const receipt = await transfer.wait()

    // The expected fee paid is the value returned by eth_estimateGas
    const gasLimit = await ERC20.estimateGas.transfer(other.address, 100)
    const gasPrice = await wallet.getGasPrice()
    expect(gasPrice).to.deep.equal(TxGasPrice)
    const expectedFeePaid = gasLimit.mul(gasPrice)

    // There are two events from the transfer with the first being
    // the ETH fee paid and the second of the value transfered (100)
    expect(receipt.events.length).to.equal(2)
    expect(receipt.events[0].args._value).to.deep.equal(expectedFeePaid)
    expect(receipt.events[1].args._from).to.equal(wallet.address)
    expect(receipt.events[1].args._value.toNumber()).to.equal(100)

    const receiverBalance = await ERC20.balanceOf(other.address)
    const senderBalance = await ERC20.balanceOf(wallet.address)

    expect(receiverBalance.toNumber()).to.equal(100)
    expect(senderBalance.toNumber()).to.equal(900)
  })

  it('should revert if trying to transfer too much', async () => {
    await expect(
      ERC20.transfer(other.address, initialAmount * 2)
    ).to.be.revertedWith('insufficient balance')
  })
})
