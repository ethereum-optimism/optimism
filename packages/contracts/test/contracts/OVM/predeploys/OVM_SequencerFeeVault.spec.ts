import { expect } from '../../../setup'

/* Imports: External */
import hre from 'hardhat'
import { MockContract, smockit } from '@eth-optimism/smock'
import { Contract, Signer, constants } from 'ethers'

/* Imports: Internal */
import { predeploys } from '../../../../src'
import { getContractFactory } from '@nomiclabs/hardhat-ethers/types'

describe('OVM_SequencerFeeVault', () => {
  let signer1: Signer
  before(async () => {
    ;[signer1] = await hre.ethers.getSigners()
  })

  let Mock__OVM_ETH: MockContract
  let Mock__OVM_L2StandardBridge: MockContract
  before(async () => {
    Mock__OVM_ETH = await smockit('OVM_ETH', {
      address: predeploys.OVM_ETH,
    })
    Mock__OVM_L2StandardBridge = await smockit('OVM_L2StandardBridge', {
      address: predeploys.OVM_L2StandardBridge,
    })
    console.log(await Mock__OVM_L2StandardBridge.getAddress)
  })

  let OVM_SequencerFeeVault: Contract
  beforeEach(async () => {
    const factory = await hre.ethers.getContractFactory('OVM_SequencerFeeVault')
    OVM_SequencerFeeVault = await factory.deploy(await signer1.getAddress())
  })

  describe('withdraw', async () => {
    it('should fail if the contract does not have more than the minimum balance', async () => {
      Mock__OVM_ETH.smocked.balanceOf.will.return.with(0)

      await expect(OVM_SequencerFeeVault.withdraw()).to.be.reverted
    })

    it('should succeed when the contract has exactly sufficient balance', async () => {
      const amount = await OVM_SequencerFeeVault.MIN_WITHDRAWAL_AMOUNT()
      Mock__OVM_ETH.smocked.balanceOf.will.return.with(amount)

      await expect(OVM_SequencerFeeVault.withdraw()).to.not.be.reverted

      expect(
        Mock__OVM_L2StandardBridge.smocked.withdrawTo.calls[0]
      ).to.deep.equal([
        predeploys.OVM_ETH,
        await signer1.getAddress(),
        amount,
        0,
        '0x',
      ])
    })

    it('should succeed when the contract has more than sufficient balance', async () => {
      const amount = hre.ethers.utils.parseEther('100')
      Mock__OVM_ETH.smocked.balanceOf.will.return.with(amount)

      await expect(OVM_SequencerFeeVault.withdraw()).to.not.be.reverted

      expect(
        Mock__OVM_L2StandardBridge.smocked.withdrawTo.calls[0]
      ).to.deep.equal([
        predeploys.OVM_ETH,
        await signer1.getAddress(),
        amount,
        0,
        '0x',
      ])
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
