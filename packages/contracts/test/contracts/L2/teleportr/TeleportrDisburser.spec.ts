/* External Imports */
import { ethers } from 'hardhat'
import { Signer, Contract, BigNumber } from 'ethers'

/* Internal Imports */
import { expect } from '../../../setup'

describe('TeleportrDisburser', async () => {
  const zeroETH = ethers.utils.parseEther('0.0')
  const oneETH = ethers.utils.parseEther('1.0')
  const twoETH = ethers.utils.parseEther('2.0')

  let teleportrDisburser: Contract
  let failingReceiver: Contract
  let signer: Signer
  let signer2: Signer
  let contractAddress: string
  let failingReceiverAddress: string
  let signerAddress: string
  let signer2Address: string
  before(async () => {
    ;[signer, signer2] = await ethers.getSigners()
    teleportrDisburser = await (
      await ethers.getContractFactory('TeleportrDisburser')
    ).deploy()
    failingReceiver = await (
      await ethers.getContractFactory('FailingReceiver')
    ).deploy()
    contractAddress = teleportrDisburser.address
    failingReceiverAddress = failingReceiver.address
    signerAddress = await signer.getAddress()
    signer2Address = await signer2.getAddress()
  })
  describe('disburse checks', async () => {
    it('should revert if called by non-owner', async () => {
      await expect(
        teleportrDisburser.connect(signer2).disburse(0, [], { value: oneETH })
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })
    it('should revert if no disbursements is zero length', async () => {
      await expect(
        teleportrDisburser.disburse(0, [], { value: oneETH })
      ).to.be.revertedWith('No disbursements')
    })
    it('should revert if nextDepositId does not match expected value', async () => {
      await expect(
        teleportrDisburser.disburse(1, [[oneETH, signer2Address]], {
          value: oneETH,
        })
      ).to.be.revertedWith('Unexpected next deposit id')
    })
    it('should revert if msg.value does not match total to disburse', async () => {
      await expect(
        teleportrDisburser.disburse(0, [[oneETH, signer2Address]], {
          value: zeroETH,
        })
      ).to.be.revertedWith('Disbursement total != amount sent')
    })
  })
  describe('disburse single success', async () => {
    let signerInitialBalance: BigNumber
    let signer2InitialBalance: BigNumber
    it('should emit DisbursementSuccess for successful disbursement', async () => {
      signerInitialBalance = await ethers.provider.getBalance(signerAddress)
      signer2InitialBalance = await ethers.provider.getBalance(signer2Address)
      await expect(
        teleportrDisburser.disburse(0, [[oneETH, signer2Address]], {
          value: oneETH,
        })
      )
        .to.emit(teleportrDisburser, 'DisbursementSuccess')
        .withArgs(BigNumber.from(0), signer2Address, oneETH)
    })
    it('should show one total disbursement', async () => {
      await expect(await teleportrDisburser.totalDisbursements()).to.be.equal(
        BigNumber.from(1)
      )
    })
    it('should leave contract balance at zero ETH', async () => {
      await expect(
        await ethers.provider.getBalance(contractAddress)
      ).to.be.equal(zeroETH)
    })
    it('should increase recipients balance by disbursement amount', async () => {
      await expect(
        await ethers.provider.getBalance(signer2Address)
      ).to.be.equal(signer2InitialBalance.add(oneETH))
    })
    it('should decrease owners balance by disbursement amount - fees', async () => {
      await expect(
        await ethers.provider.getBalance(signerAddress)
      ).to.be.closeTo(signerInitialBalance.sub(oneETH), 10 ** 15)
    })
  })
  describe('disburse single failure', async () => {
    let signerInitialBalance: BigNumber
    it('should emit DisbursementFailed for failed disbursement', async () => {
      signerInitialBalance = await ethers.provider.getBalance(signerAddress)
      await expect(
        teleportrDisburser.disburse(1, [[oneETH, failingReceiverAddress]], {
          value: oneETH,
        })
      )
        .to.emit(teleportrDisburser, 'DisbursementFailed')
        .withArgs(BigNumber.from(1), failingReceiverAddress, oneETH)
    })
    it('should show two total disbursements', async () => {
      await expect(await teleportrDisburser.totalDisbursements()).to.be.equal(
        BigNumber.from(2)
      )
    })
    it('should leave contract with disbursement amount', async () => {
      await expect(
        await ethers.provider.getBalance(contractAddress)
      ).to.be.equal(oneETH)
    })
    it('should leave recipients balance at zero ETH', async () => {
      await expect(
        await ethers.provider.getBalance(failingReceiverAddress)
      ).to.be.equal(zeroETH)
    })
    it('should decrease owners balance by disbursement amount - fees', async () => {
      await expect(
        await ethers.provider.getBalance(signerAddress)
      ).to.be.closeTo(signerInitialBalance.sub(oneETH), 10 ** 15)
    })
  })
  describe('withdrawBalance', async () => {
    let initialContractBalance: BigNumber
    let initialSignerBalance: BigNumber
    it('should revert if called by non-owner', async () => {
      await expect(
        teleportrDisburser.connect(signer2).withdrawBalance()
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })
    it('should emit BalanceWithdrawn if called by owner', async () => {
      initialContractBalance = await ethers.provider.getBalance(contractAddress)
      initialSignerBalance = await ethers.provider.getBalance(signerAddress)
      await expect(teleportrDisburser.withdrawBalance())
        .to.emit(teleportrDisburser, 'BalanceWithdrawn')
        .withArgs(signerAddress, oneETH)
    })
    it('should leave contract with zero balance', async () => {
      await expect(await ethers.provider.getBalance(contractAddress)).to.equal(
        zeroETH
      )
    })
    it('should credit owner with contract balance - fees', async () => {
      const expSignerBalance = initialSignerBalance.add(initialContractBalance)
      await expect(
        await ethers.provider.getBalance(signerAddress)
      ).to.be.closeTo(expSignerBalance, 10 ** 15)
    })
  })
  describe('disburse multiple', async () => {
    let signerInitialBalance: BigNumber
    let signer2InitialBalance: BigNumber
    it('should emit DisbursementSuccess for successful disbursement', async () => {
      signerInitialBalance = await ethers.provider.getBalance(signerAddress)
      signer2InitialBalance = await ethers.provider.getBalance(signer2Address)
      await expect(
        teleportrDisburser.disburse(
          2,
          [
            [oneETH, signer2Address],
            [oneETH, failingReceiverAddress],
          ],
          { value: twoETH }
        )
      ).to.not.be.reverted
    })
    it('should show four total disbursements', async () => {
      await expect(await teleportrDisburser.totalDisbursements()).to.be.equal(
        BigNumber.from(4)
      )
    })
    it('should leave contract balance with failed disbursement amount', async () => {
      await expect(
        await ethers.provider.getBalance(contractAddress)
      ).to.be.equal(oneETH)
    })
    it('should increase success recipients balance by disbursement amount', async () => {
      await expect(
        await ethers.provider.getBalance(signer2Address)
      ).to.be.equal(signer2InitialBalance.add(oneETH))
    })
    it('should leave failed recipients balance at zero ETH', async () => {
      await expect(
        await ethers.provider.getBalance(failingReceiverAddress)
      ).to.be.equal(zeroETH)
    })
    it('should decrease owners balance by disbursement 2*amount - fees', async () => {
      await expect(
        await ethers.provider.getBalance(signerAddress)
      ).to.be.closeTo(signerInitialBalance.sub(twoETH), 10 ** 15)
    })
  })
})
