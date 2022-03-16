/* Imports: External */
import hre from 'hardhat'
import { smock, FakeContract } from '@defi-wonderland/smock'
import { Contract, Signer } from 'ethers'

/* Imports: Internal */
import { expect } from '../../../setup'
import { predeploys } from '../../../../src'

describe('OVM_SequencerFeeVault', () => {
  let signer1: Signer
  before(async () => {
    ;[signer1] = await hre.ethers.getSigners()
  })

  let Fake__L2StandardBridge: FakeContract
  before(async () => {
    Fake__L2StandardBridge = await smock.fake<Contract>('L2StandardBridge', {
      address: predeploys.L2StandardBridge,
    })
  })

  let OVM_SequencerFeeVault: Contract
  beforeEach(async () => {
    const factory = await hre.ethers.getContractFactory('OVM_SequencerFeeVault')
    OVM_SequencerFeeVault = await factory.deploy(await signer1.getAddress())
  })

  describe('withdraw', async () => {
    it('should fail if the contract does not have more than the minimum balance', async () => {
      await expect(OVM_SequencerFeeVault.withdraw()).to.be.reverted
    })

    it('should succeed when the contract has exactly sufficient balance', async () => {
      // Send just the balance that the contract needs.
      const amount = await OVM_SequencerFeeVault.MIN_WITHDRAWAL_AMOUNT()

      await signer1.sendTransaction({
        to: OVM_SequencerFeeVault.address,
        value: amount,
      })

      await expect(OVM_SequencerFeeVault.withdraw()).to.not.be.reverted

      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(0).args[0]
      ).to.deep.equal(predeploys.OVM_ETH)
      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(0).args[1]
      ).to.deep.equal(await signer1.getAddress())
      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(0).args[2]
      ).to.deep.equal(amount)
      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(0).args[3]
      ).to.deep.equal(0)
      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(0).args[4]
      ).to.deep.equal('0x')
    })

    it('should succeed when the contract has more than sufficient balance', async () => {
      // Send just twice the balance that the contract needs.
      let amount = await OVM_SequencerFeeVault.MIN_WITHDRAWAL_AMOUNT()
      amount = amount.mul(2)

      await signer1.sendTransaction({
        to: OVM_SequencerFeeVault.address,
        value: amount,
      })

      await expect(OVM_SequencerFeeVault.withdraw()).to.not.be.reverted
      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(1).args[0]
      ).to.deep.equal(predeploys.OVM_ETH)
      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(1).args[1]
      ).to.deep.equal(await signer1.getAddress())
      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(1).args[2]
      ).to.deep.equal(amount)
      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(1).args[3]
      ).to.deep.equal(0)
      expect(
        Fake__L2StandardBridge.withdrawTo.getCall(1).args[4]
      ).to.deep.equal('0x')
    })

    it('should have an owner in storage slot 0x00...00', async () => {
      // Deploy a new temporary instance with an address that's easier to make assertions about.
      const factory = await hre.ethers.getContractFactory(
        'OVM_SequencerFeeVault'
      )
      OVM_SequencerFeeVault = await factory.deploy(`0x${'11'.repeat(20)}`)

      expect(
        await hre.ethers.provider.getStorageAt(
          OVM_SequencerFeeVault.address,
          hre.ethers.constants.HashZero
        )
      ).to.equal(`0x000000000000000000000000${'11'.repeat(20)}`)
    })
  })
})
