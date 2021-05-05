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
  const useL2 = process.env.TARGET === 'OVM'

  if (useL2 == true) {
    provider = new ethers.providers.JsonRpcProvider('http://127.0.0.1:8545')
    provider.pollingInterval = 100
    provider.getGasPrice = async () => ethers.BigNumber.from(0)
  } else {
    provider = new ethers.providers.JsonRpcProvider('http://127.0.0.1:9545')
  }

  walletTo = new ethers.Wallet(privateKey, provider)
  walletEmpty = new ethers.Wallet(privateKeyEmpty, provider)

  // parameters to use for our test coin
  const COIN_NAME = 'OVM Test Coin'
  const TICKER = 'OVM'
  const NUM_DECIMALS = 1

  describe('when using a deployed contract instance', () => {
    before(async () => {
      wallet = await provider.getSigner(0)
      walletAddress = await wallet.getAddress()
      walletToAddress = await walletTo.getAddress()
      walletEmptyAddress = await walletEmpty.getAddress()

      const Artifact__ERC20 = getArtifact(useL2)
      const Factory__ERC20 = new ethers.ContractFactory(
        Artifact__ERC20.abi,
        Artifact__ERC20.bytecode,
        wallet
      )

      // TODO: Remove this hardcoded gas limit
      ERC20 = await Factory__ERC20.connect(wallet).deploy(
        1000,
        COIN_NAME,
        NUM_DECIMALS,
        TICKER
      )
      await ERC20.deployTransaction.wait()
    })

    it('should assigns initial balance', async () => {
      expect(await ERC20.balanceOf(walletAddress)).to.equal(1000)
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
      await expect(ERC20.transfer(walletToAddress, 1007)).to.be.reverted
      const walletToBalanceAfter = await ERC20.balanceOf(walletToAddress)
      expect(walletToBalanceBefore).to.eq(walletToBalanceAfter)
    })

    it('should not transfer from empty account', async () => {
      const ERC20FromOtherWallet = ERC20.connect(walletEmpty)
      await expect(ERC20FromOtherWallet.transfer(walletEmptyAddress, 1)).to.be
        .reverted
    })
  })
})
