/* External Imports */
import { ethers } from 'hardhat'
import { Signer, Contract, BigNumber } from 'ethers'

/* Internal Imports */
import { expect } from '../../../setup'

const initialMinDepositAmount = ethers.utils.parseEther('0.01')
const initialMaxDepositAmount = ethers.utils.parseEther('1')
const initialMaxBalance = ethers.utils.parseEther('2')

describe('TeleportrDeposit', async () => {
  let teleportrDeposit: Contract
  let signer: Signer
  let signer2: Signer
  let contractAddress: string
  let signerAddress: string
  let signer2Address: string
  before(async () => {
    ;[signer, signer2] = await ethers.getSigners()
    teleportrDeposit = await (
      await ethers.getContractFactory('TeleportrDeposit')
    ).deploy(
      initialMinDepositAmount,
      initialMaxDepositAmount,
      initialMaxBalance
    )
    contractAddress = teleportrDeposit.address
    signerAddress = await signer.getAddress()
    signer2Address = await signer2.getAddress()
  })
  describe('receive', async () => {
    const oneETH = ethers.utils.parseEther('1.0')
    const twoETH = ethers.utils.parseEther('2.0')
    it('should revert if deposit amount is less than min amount', async () => {
      await expect(
        signer.sendTransaction({
          to: contractAddress,
          value: ethers.utils.parseEther('0.001'),
        })
      ).to.be.revertedWith('Deposit amount is too small')
    })
    it('should revert if deposit amount is greater than max amount', async () => {
      await expect(
        signer.sendTransaction({
          to: contractAddress,
          value: ethers.utils.parseEther('1.1'),
        })
      ).to.be.revertedWith('Deposit amount is too big')
    })
    it('should emit EtherReceived if called by non-owner', async () => {
      await expect(
        signer2.sendTransaction({
          to: contractAddress,
          value: oneETH,
        })
      )
        .to.emit(teleportrDeposit, 'EtherReceived')
        .withArgs(BigNumber.from('0'), signer2Address, oneETH)
    })
    it('should increase the contract balance by deposit amount', async () => {
      await expect(await ethers.provider.getBalance(contractAddress)).to.equal(
        oneETH
      )
    })
    it('should emit EtherReceived if called by owner', async () => {
      await expect(
        signer.sendTransaction({
          to: contractAddress,
          value: oneETH,
        })
      )
        .to.emit(teleportrDeposit, 'EtherReceived')
        .withArgs(BigNumber.from('1'), signerAddress, oneETH)
    })
    it('should increase the contract balance by deposit amount', async () => {
      await expect(await ethers.provider.getBalance(contractAddress)).to.equal(
        twoETH
      )
    })
    it('should revert if deposit will exceed max balance', async () => {
      await expect(
        signer.sendTransaction({
          to: contractAddress,
          value: initialMinDepositAmount,
        })
      ).to.be.revertedWith('Contract max balance exceeded')
    })
  })
  describe('withdrawBalance', async () => {
    let initialContractBalance: BigNumber
    let initialSignerBalance: BigNumber
    before(async () => {
      initialContractBalance = await ethers.provider.getBalance(contractAddress)
      initialSignerBalance = await ethers.provider.getBalance(signerAddress)
    })
    it('should revert if called by non-owner', async () => {
      await expect(
        teleportrDeposit.connect(signer2).withdrawBalance()
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })
    it('should emit BalanceWithdrawn if called by owner', async () => {
      await expect(teleportrDeposit.withdrawBalance())
        .to.emit(teleportrDeposit, 'BalanceWithdrawn')
        .withArgs(signerAddress, initialContractBalance)
    })
    it('should leave the contract with zero balance', async () => {
      await expect(await ethers.provider.getBalance(contractAddress)).to.equal(
        ethers.utils.parseEther('0')
      )
    })
    it('should credit owner with contract balance - fees', async () => {
      const expSignerBalance = initialSignerBalance.add(initialContractBalance)
      await expect(
        await ethers.provider.getBalance(signerAddress)
      ).to.be.closeTo(expSignerBalance, 10 ** 15)
    })
  })
  describe('setMinAmount', async () => {
    const newMinDepositAmount = ethers.utils.parseEther('0.02')
    it('should revert if called by non-owner', async () => {
      await expect(
        teleportrDeposit.connect(signer2).setMinAmount(newMinDepositAmount)
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })
    it('should emit MinDepositAmountSet if called by owner', async () => {
      await expect(teleportrDeposit.setMinAmount(newMinDepositAmount))
        .to.emit(teleportrDeposit, 'MinDepositAmountSet')
        .withArgs(initialMinDepositAmount, newMinDepositAmount)
    })
    it('should have updated minDepositAmount after success', async () => {
      await expect(await teleportrDeposit.minDepositAmount()).to.be.eq(
        newMinDepositAmount
      )
    })
  })
  describe('setMaxAmount', async () => {
    const newMaxDepositAmount = ethers.utils.parseEther('2')
    it('should revert if called non-owner', async () => {
      await expect(
        teleportrDeposit.connect(signer2).setMaxAmount(newMaxDepositAmount)
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })
    it('should emit MaxDepositAmountSet if called by owner', async () => {
      await expect(teleportrDeposit.setMaxAmount(newMaxDepositAmount))
        .to.emit(teleportrDeposit, 'MaxDepositAmountSet')
        .withArgs(initialMaxDepositAmount, newMaxDepositAmount)
    })
    it('should have an updated maxDepositAmount after success', async () => {
      await expect(await teleportrDeposit.maxDepositAmount()).to.be.eq(
        newMaxDepositAmount
      )
    })
  })
  describe('setMaxBalance', async () => {
    const newMaxBalance = ethers.utils.parseEther('2000')
    it('should revert if called by non-owner', async () => {
      await expect(
        teleportrDeposit.connect(signer2).setMaxBalance(newMaxBalance)
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })
    it('should emit MaxBalanceSet if called by owner', async () => {
      await expect(teleportrDeposit.setMaxBalance(newMaxBalance))
        .to.emit(teleportrDeposit, 'MaxBalanceSet')
        .withArgs(initialMaxBalance, newMaxBalance)
    })
    it('should have an updated maxBalance after success', async () => {
      await expect(await teleportrDeposit.maxBalance()).to.be.eq(newMaxBalance)
    })
  })
})
