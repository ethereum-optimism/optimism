import { ethers } from 'hardhat'
import { Contract, BigNumber } from 'ethers'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../setup'
import { deploy } from '../../helpers'

const zeroETH = ethers.utils.parseEther('0.0')
const oneETH = ethers.utils.parseEther('1.0')
const twoETH = ethers.utils.parseEther('2.0')

describe('TeleportrDisburser', async () => {
  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let TeleportrDisburser: Contract
  let FailingReceiver: Contract
  before(async () => {
    TeleportrDisburser = await deploy('TeleportrDisburser')
    FailingReceiver = await deploy('FailingReceiver')
  })

  describe('disburse checks', async () => {
    it('should revert if called by non-owner', async () => {
      await expect(
        TeleportrDisburser.connect(signer2).disburse(0, [], { value: oneETH })
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })

    it('should revert if no disbursements is zero length', async () => {
      await expect(
        TeleportrDisburser.disburse(0, [], { value: oneETH })
      ).to.be.revertedWith('No disbursements')
    })

    it('should revert if nextDepositId does not match expected value', async () => {
      await expect(
        TeleportrDisburser.disburse(1, [[oneETH, signer2.address]], {
          value: oneETH,
        })
      ).to.be.revertedWith('Unexpected next deposit id')
    })

    it('should revert if msg.value does not match total to disburse', async () => {
      await expect(
        TeleportrDisburser.disburse(0, [[oneETH, signer2.address]], {
          value: zeroETH,
        })
      ).to.be.revertedWith('Disbursement total != amount sent')
    })
  })

  describe('disburse single success', async () => {
    let signer1InitialBalance: BigNumber
    let signer2InitialBalance: BigNumber
    before(async () => {
      signer1InitialBalance = await ethers.provider.getBalance(signer1.address)
      signer2InitialBalance = await ethers.provider.getBalance(signer2.address)
    })

    it('should emit DisbursementSuccess for successful disbursement', async () => {
      await expect(
        TeleportrDisburser.disburse(0, [[oneETH, signer2.address]], {
          value: oneETH,
        })
      )
        .to.emit(TeleportrDisburser, 'DisbursementSuccess')
        .withArgs(BigNumber.from(0), signer2.address, oneETH)
    })

    it('should show one total disbursement', async () => {
      expect(await TeleportrDisburser.totalDisbursements()).to.be.equal(
        BigNumber.from(1)
      )
    })

    it('should leave contract balance at zero ETH', async () => {
      expect(
        await ethers.provider.getBalance(TeleportrDisburser.address)
      ).to.be.equal(zeroETH)
    })

    it('should increase recipients balance by disbursement amount', async () => {
      expect(await ethers.provider.getBalance(signer2.address)).to.be.equal(
        signer2InitialBalance.add(oneETH)
      )
    })

    it('should decrease owners balance by disbursement amount - fees', async () => {
      expect(await ethers.provider.getBalance(signer1.address)).to.be.closeTo(
        signer1InitialBalance.sub(oneETH),
        10 ** 15
      )
    })
  })

  describe('disburse single failure', async () => {
    let signer1InitialBalance: BigNumber
    before(async () => {
      signer1InitialBalance = await ethers.provider.getBalance(signer1.address)
    })

    it('should emit DisbursementFailed for failed disbursement', async () => {
      await expect(
        TeleportrDisburser.disburse(1, [[oneETH, FailingReceiver.address]], {
          value: oneETH,
        })
      )
        .to.emit(TeleportrDisburser, 'DisbursementFailed')
        .withArgs(BigNumber.from(1), FailingReceiver.address, oneETH)
    })

    it('should show two total disbursements', async () => {
      expect(await TeleportrDisburser.totalDisbursements()).to.be.equal(
        BigNumber.from(2)
      )
    })

    it('should leave contract with disbursement amount', async () => {
      expect(
        await ethers.provider.getBalance(TeleportrDisburser.address)
      ).to.be.equal(oneETH)
    })

    it('should leave recipients balance at zero ETH', async () => {
      expect(
        await ethers.provider.getBalance(FailingReceiver.address)
      ).to.be.equal(zeroETH)
    })

    it('should decrease owners balance by disbursement amount - fees', async () => {
      expect(await ethers.provider.getBalance(signer1.address)).to.be.closeTo(
        signer1InitialBalance.sub(oneETH),
        10 ** 15
      )
    })
  })

  describe('withdrawBalance', async () => {
    let signer1InitialBalance: BigNumber
    let disburserInitialBalance: BigNumber
    before(async () => {
      signer1InitialBalance = await ethers.provider.getBalance(signer1.address)
      disburserInitialBalance = await ethers.provider.getBalance(
        TeleportrDisburser.address
      )
    })

    it('should revert if called by non-owner', async () => {
      await expect(
        TeleportrDisburser.connect(signer2).withdrawBalance()
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })

    it('should emit BalanceWithdrawn if called by owner', async () => {
      await expect(TeleportrDisburser.withdrawBalance())
        .to.emit(TeleportrDisburser, 'BalanceWithdrawn')
        .withArgs(signer1.address, oneETH)
    })

    it('should leave contract with zero balance', async () => {
      expect(
        await ethers.provider.getBalance(TeleportrDisburser.address)
      ).to.equal(zeroETH)
    })

    it('should credit owner with contract balance - fees', async () => {
      expect(await ethers.provider.getBalance(signer1.address)).to.be.closeTo(
        signer1InitialBalance.add(disburserInitialBalance),
        10 ** 15
      )
    })
  })

  describe('disburse multiple', async () => {
    let signer1InitialBalance: BigNumber
    let signer2InitialBalance: BigNumber
    before(async () => {
      signer1InitialBalance = await ethers.provider.getBalance(signer1.address)
      signer2InitialBalance = await ethers.provider.getBalance(signer2.address)
    })

    it('should emit DisbursementSuccess for successful disbursement', async () => {
      await expect(
        TeleportrDisburser.disburse(
          2,
          [
            [oneETH, signer2.address],
            [oneETH, FailingReceiver.address],
          ],
          { value: twoETH }
        )
      ).to.not.be.reverted
    })

    it('should show four total disbursements', async () => {
      expect(await TeleportrDisburser.totalDisbursements()).to.be.equal(
        BigNumber.from(4)
      )
    })

    it('should leave contract balance with failed disbursement amount', async () => {
      expect(
        await ethers.provider.getBalance(TeleportrDisburser.address)
      ).to.be.equal(oneETH)
    })

    it('should increase success recipients balance by disbursement amount', async () => {
      expect(await ethers.provider.getBalance(signer2.address)).to.be.equal(
        signer2InitialBalance.add(oneETH)
      )
    })

    it('should leave failed recipients balance at zero ETH', async () => {
      expect(
        await ethers.provider.getBalance(FailingReceiver.address)
      ).to.be.equal(zeroETH)
    })

    it('should decrease owners balance by disbursement 2*amount - fees', async () => {
      expect(await ethers.provider.getBalance(signer1.address)).to.be.closeTo(
        signer1InitialBalance.sub(twoETH),
        10 ** 15
      )
    })
  })
})
