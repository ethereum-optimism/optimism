import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, Contract } from 'ethers'

/* Internal Imports */
import { makeAddressManager } from '../../../helpers'

describe('OVM_BondManager', () => {
  let sequencer: Signer
  let nonSequencer: Signer
  before(async () => {
    ;[sequencer, nonSequencer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let OVM_BondManager: Contract
  before(async () => {
    OVM_BondManager = await (
      await ethers.getContractFactory('OVM_BondManager')
    ).deploy(AddressManager.address)

    AddressManager.setAddress('OVM_Proposer', await sequencer.getAddress())
  })

  describe('isCollateralized', () => {
    it('should return true for OVM_Proposer', async () => {
      expect(
        await OVM_BondManager.isCollateralized(await sequencer.getAddress())
      ).to.equal(true)
    })

    it('should return false for non-sequencer', async () => {
      expect(
        await OVM_BondManager.isCollateralized(await nonSequencer.getAddress())
      ).to.equal(false)
    })
  })
})
