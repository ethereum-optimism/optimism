import { expect } from '../../../setup'

/* Imports: External */
import hre from 'hardhat'
import { MockContract, smockit } from '@eth-optimism/smock'
import { Contract } from 'ethers'

/* Imports: Internal */
import { predeploys } from '../../../../src'

describe('OVM_SequencerFeeVault', () => {
  let Mock__OVM_ETH: MockContract
  before(async () => {
    Mock__OVM_ETH = await smockit('OVM_ETH', {
      address: predeploys.OVM_ETH,
    })
  })

  let OVM_SequencerFeeVault: Contract
  beforeEach(async () => {
    const factory = await hre.ethers.getContractFactory('OVM_SequencerFeeVault')
    OVM_SequencerFeeVault = await factory.deploy()
  })

  describe('withdraw', async () => {
    it('should fail if the contract does not have more than the minimum balance', async () => {
      Mock__OVM_ETH.smocked.balanceOf.will.return.with(0)

      await expect(OVM_SequencerFeeVault.withdraw()).to.be.reverted
    })

    it('should succeed when the contract has exactly sufficient balance', async () => {
      const amount = hre.ethers.utils.parseEther('10')
      Mock__OVM_ETH.smocked.balanceOf.will.return.with(amount)

      await expect(OVM_SequencerFeeVault.withdraw()).to.not.be.reverted

      expect(Mock__OVM_ETH.smocked.withdrawTo.calls[0]).to.deep.equal([
        hre.ethers.constants.AddressZero,
        amount,
        0,
        '0x',
      ])
    })

    it('should succeed when the contract has more than sufficient balance', async () => {
      const amount = hre.ethers.utils.parseEther('100')
      Mock__OVM_ETH.smocked.balanceOf.will.return.with(amount)

      await expect(OVM_SequencerFeeVault.withdraw()).to.not.be.reverted

      expect(Mock__OVM_ETH.smocked.withdrawTo.calls[0]).to.deep.equal([
        hre.ethers.constants.AddressZero,
        amount,
        0,
        '0x',
      ])
    })
  })
})
