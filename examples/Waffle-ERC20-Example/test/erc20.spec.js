/* External imports */
require('dotenv/config')
const { use, expect } = require('chai')
const { ethers } = require('ethers')
const { solidity } = require('ethereum-waffle')

/* Internal imports */
const { getArtifact } = require('./getArtifact')

use(solidity)

describe('ERC20 smart contract', () => {
  let ERC20,
    provider,
    wallet,
    walletTo,
    walletEmpty,
    walletAddress,
    walletToAddress,
    walletEmptyAddress

  const privateKey = ethers.Wallet.createRandom().privateKey
  const privateKeyEmpty = ethers.Wallet.createRandom().privateKey
  const useL2 = (process.env.TARGET === 'OVM')

  if (useL2 == true) {
    provider = new ethers.providers.JsonRpcProvider('http://0.0.0.0:8545')
  } else {
    provider = new ethers.providers.JsonRpcProvider('http://0.0.0.0:9545')
  }

  wallet = new ethers.Wallet(
    '0x754fde3f5e60ef2c7649061e06957c29017fe21032a8017132c0078e37f6193a',
    provider
  )
  walletTo = new ethers.Wallet(privateKey, provider)
  walletEmpty = new ethers.Wallet(privateKeyEmpty, provider)

  // parameters to use for our test coin
  const COIN_NAME = 'OVM Test Coin'
  const TICKER = 'OVM'
  const NUM_DECIMALS = 1


  describe('when using a deployed contract instance', () => {
    before(async () => {
      walletAddress = await wallet.getAddress()
      walletToAddress = await walletTo.getAddress()
      walletEmptyAddress = await walletEmpty.getAddress()

      const Artifact__ERC20 = getArtifact(process.env.TARGET)
      const Factory__ERC20 = new ethers.ContractFactory(
        Artifact__ERC20.abi,
        Artifact__ERC20.bytecode,
        wallet
      )

      ERC20 = await Factory__ERC20
        .connect(wallet)
        .deploy(1000, COIN_NAME, NUM_DECIMALS, TICKER)

      ERC20.deployTransaction.wait()
    })

    it('should assigns initial balance', async () => {
      expect(await ERC20.balanceOf(wallet.address)).to.equal(1000)
    })

    it('should correctly set vanity information', async () => {
      const name = await ERC20.name()
      expect(name).to.equal(COIN_NAME)

      const decimals = await ERC20.decimals()
      expect(decimals).to.equal(NUM_DECIMALS)

      const symbol = await ERC20.symbol()
      expect(symbol).to.equal(TICKER)
    })


    it('should transfer amount to destination account', async () => {
      const tx = await ERC20.connect(wallet).transfer(walletToAddress, 7)
      await tx.wait()
      const walletToBalance = await ERC20.balanceOf(walletToAddress)
      expect(walletToBalance.toString()).to.equal('7')
    })

    it('should emit Transfer event', async () => {
      const tx = ERC20.connect(wallet).transfer(walletToAddress, 7)
      await expect(tx)
        .to.emit(ERC20, 'Transfer')
        .withArgs(walletAddress, walletToAddress, 7)
    })

    it('should not transfer above the amount', async () => {
      const walletToBalanceBefore = await ERC20.balanceOf(walletToAddress)
      const tx = await ERC20.transfer(walletToAddress, 1007)
      const walletToBalanceAfter = await ERC20.balanceOf(walletToAddress)
      expect(walletToBalanceBefore).to.eq(walletToBalanceAfter)
    })

    it('should not transfer from empty account', async () => {
      if (useL2 == true) {
        const walletToBalanceBefore = await ERC20.balanceOf(walletEmptyAddress)
        const ERC20FromOtherWallet = ERC20.connect(walletEmpty)
        const tx = await ERC20FromOtherWallet.transfer(walletEmptyAddress, 1)
        const walletToBalanceAfter = await ERC20.balanceOf(walletEmptyAddress)
        expect(walletToBalanceBefore).to.eq(walletToBalanceAfter)
      } else {
        const ERC20FromOtherWallet = ERC20.connect(walletTo)
        const tx = ERC20FromOtherWallet.transfer(walletAddress, 1)
        await expect(tx).to.be.reverted
      }
    })
  })
})