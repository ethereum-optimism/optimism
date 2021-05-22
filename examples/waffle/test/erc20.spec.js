/* External imports */
require('dotenv/config')
const { use, expect } = require('chai')
const { ethers } = require('ethers')
const { solidity } = require('ethereum-waffle')

/* Internal imports */
const { getArtifact } = require('./getArtifact')

use(solidity)

const config = {
  l2Url: process.env.L2_URL || 'http://127.0.0.1:8545',
  l1Url: process.env.L1_URL || 'http://127.0.0.1:9545',
  useL2: process.env.TARGET === 'OVM',
  privateKey: process.env.PRIVATE_KEY || '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80'
}

describe('ERC20 smart contract', () => {
  let ERC20,
    provider

  if (config.useL2) {
    provider = new ethers.providers.JsonRpcProvider(config.l2Url)
    provider.pollingInterval = 100
    provider.getGasPrice = async () => ethers.BigNumber.from(0)
  } else {
    provider = new ethers.providers.JsonRpcProvider(config.l1Url)
  }

  const wallet = new ethers.Wallet(config.privateKey).connect(provider)

  // parameters to use for our test coin
  const COIN_NAME = 'OVM Test Coin'
  const TICKER = 'OVM'
  const NUM_DECIMALS = 1

  describe('when using a deployed contract instance', () => {
    before(async () => {
      const Artifact__ERC20 = getArtifact(config.useL2)
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
      const address = await wallet.getAddress()
      expect(await ERC20.balanceOf(address)).to.equal(1000)
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
      const freshWallet = ethers.Wallet.createRandom()
      const destination = await freshWallet.getAddress()
      const tx = await ERC20.connect(wallet).transfer(destination, 7)
      await tx.wait()
      const walletToBalance = await ERC20.balanceOf(destination)
      expect(walletToBalance.toString()).to.equal('7')
    })

    it('should emit Transfer event', async () => {
      const address = await wallet.getAddress()
      const tx = ERC20.connect(wallet).transfer(address, 7)
      await expect(tx)
        .to.emit(ERC20, 'Transfer')
        .withArgs(address, address, 7)
    })

    it('should not transfer above the amount', async () => {
      const address = await wallet.getAddress()
      const walletToBalanceBefore = await ERC20.balanceOf(address)
      await expect(ERC20.transfer(address, 1007)).to.be.reverted
      const walletToBalanceAfter = await ERC20.balanceOf(address)
      expect(walletToBalanceBefore).to.eq(walletToBalanceAfter)
    })

    it('should not transfer from empty account', async () => {
      const emptyWallet = ethers.Wallet.createRandom()
      const address = await emptyWallet.getAddress()
      const ERC20FromOtherWallet = ERC20.connect(emptyWallet)
      await expect(ERC20FromOtherWallet.transfer(address, 1)).to.be
        .reverted
    })
  })
})
