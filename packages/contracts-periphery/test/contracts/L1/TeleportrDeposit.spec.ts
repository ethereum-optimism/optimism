import { ethers } from 'hardhat'
import { Contract, BigNumber } from 'ethers'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../setup'
import { deploy } from '../../helpers'

const initialMinDepositAmount = ethers.utils.parseEther('0.01')
const initialMaxDepositAmount = ethers.utils.parseEther('1')
const initialMaxBalance = ethers.utils.parseEther('2')

describe('TeleportrDeposit', async () => {
  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let TeleportrDeposit: Contract
  before(async () => {
    TeleportrDeposit = await deploy('TeleportrDeposit', {
      args: [
        initialMinDepositAmount,
        initialMaxDepositAmount,
        initialMaxBalance,
      ],
    })
  })

  describe('receive', async () => {
    const oneETH = ethers.utils.parseEther('1.0')
    const twoETH = ethers.utils.parseEther('2.0')

    it('should revert if deposit amount is less than min amount', async () => {
      await expect(
        signer1.sendTransaction({
          to: TeleportrDeposit.address,
          value: ethers.utils.parseEther('0.001'),
        })
      ).to.be.revertedWith('Deposit amount is too small')
    })

    it('should revert if deposit amount is greater than max amount', async () => {
      await expect(
        signer1.sendTransaction({
          to: TeleportrDeposit.address,
          value: ethers.utils.parseEther('1.1'),
        })
      ).to.be.revertedWith('Deposit amount is too big')
    })

    it('should emit EtherReceived if called by non-owner', async () => {
      await expect(
        signer2.sendTransaction({
          to: TeleportrDeposit.address,
          value: oneETH,
        })
      )
        .to.emit(TeleportrDeposit, 'EtherReceived')
        .withArgs(BigNumber.from('0'), signer2.address, oneETH)
    })

    it('should increase the contract balance by deposit amount', async () => {
      expect(
        await ethers.provider.getBalance(TeleportrDeposit.address)
      ).to.equal(oneETH)
    })

    it('should emit EtherReceived if called by owner', async () => {
      await expect(
        signer1.sendTransaction({
          to: TeleportrDeposit.address,
          value: oneETH,
        })
      )
        .to.emit(TeleportrDeposit, 'EtherReceived')
        .withArgs(BigNumber.from('1'), signer1.address, oneETH)
    })

    it('should increase the contract balance by deposit amount', async () => {
      expect(
        await ethers.provider.getBalance(TeleportrDeposit.address)
      ).to.equal(twoETH)
    })

    it('should revert if deposit will exceed max balance', async () => {
      await expect(
        signer1.sendTransaction({
          to: TeleportrDeposit.address,
          value: initialMinDepositAmount,
        })
      ).to.be.revertedWith('Contract max balance exceeded')
    })
  })

  describe('withdrawBalance', async () => {
    let initialContractBalance: BigNumber
    let initialSignerBalance: BigNumber
    before(async () => {
      initialContractBalance = await ethers.provider.getBalance(
        TeleportrDeposit.address
      )
      initialSignerBalance = await signer1.getBalance()
    })

    it('should revert if called by non-owner', async () => {
      await expect(
        TeleportrDeposit.connect(signer2).withdrawBalance()
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })

    it('should emit BalanceWithdrawn if called by owner', async () => {
      await expect(TeleportrDeposit.withdrawBalance())
        .to.emit(TeleportrDeposit, 'BalanceWithdrawn')
        .withArgs(signer1.address, initialContractBalance)
    })

    it('should leave the contract with zero balance', async () => {
      expect(
        await ethers.provider.getBalance(TeleportrDeposit.address)
      ).to.equal(ethers.utils.parseEther('0'))
    })

    it('should credit owner with contract balance - fees', async () => {
      const expSignerBalance = initialSignerBalance.add(initialContractBalance)
      expect(await signer1.getBalance()).to.be.closeTo(
        expSignerBalance,
        10 ** 15
      )
    })
  })

  describe('setMinAmount', async () => {
    const newMinDepositAmount = ethers.utils.parseEther('0.02')

    it('should revert if called by non-owner', async () => {
      await expect(
        TeleportrDeposit.connect(signer2).setMinAmount(newMinDepositAmount)
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })

    it('should emit MinDepositAmountSet if called by owner', async () => {
      await expect(TeleportrDeposit.setMinAmount(newMinDepositAmount))
        .to.emit(TeleportrDeposit, 'MinDepositAmountSet')
        .withArgs(initialMinDepositAmount, newMinDepositAmount)
    })

    it('should have updated minDepositAmount after success', async () => {
      expect(await TeleportrDeposit.minDepositAmount()).to.be.eq(
        newMinDepositAmount
      )
    })
  })

  describe('setMaxAmount', async () => {
    const newMaxDepositAmount = ethers.utils.parseEther('2')
    it('should revert if called non-owner', async () => {
      await expect(
        TeleportrDeposit.connect(signer2).setMaxAmount(newMaxDepositAmount)
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })

    it('should emit MaxDepositAmountSet if called by owner', async () => {
      await expect(TeleportrDeposit.setMaxAmount(newMaxDepositAmount))
        .to.emit(TeleportrDeposit, 'MaxDepositAmountSet')
        .withArgs(initialMaxDepositAmount, newMaxDepositAmount)
    })

    it('should have an updated maxDepositAmount after success', async () => {
      expect(await TeleportrDeposit.maxDepositAmount()).to.be.eq(
        newMaxDepositAmount
      )
    })
  })

  describe('setMaxBalance', async () => {
    const newMaxBalance = ethers.utils.parseEther('2000')

    it('should revert if called by non-owner', async () => {
      await expect(
        TeleportrDeposit.connect(signer2).setMaxBalance(newMaxBalance)
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })

    it('should emit MaxBalanceSet if called by owner', async () => {
      await expect(TeleportrDeposit.setMaxBalance(newMaxBalance))
        .to.emit(TeleportrDeposit, 'MaxBalanceSet')
        .withArgs(initialMaxBalance, newMaxBalance)
    })

    it('should have an updated maxBalance after success', async () => {
      expect(await TeleportrDeposit.maxBalance()).to.be.eq(newMaxBalance)
    })
  })
})
