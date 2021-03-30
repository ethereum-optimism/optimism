import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, Contract } from 'ethers'

/* Internal Imports */
import { makeAddressManager } from '../../../helpers'

describe('mockOVM_BondManager', () => {
  let sequencer: Signer
  let nonSequencer: Signer
  before(async () => {
    ;[sequencer, nonSequencer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let mockOVM_BondManager: Contract
  before(async () => {
    mockOVM_BondManager = await (
      await ethers.getContractFactory('mockOVM_BondManager')
    ).deploy(AddressManager.address)

    AddressManager.setAddress('OVM_Proposer', await sequencer.getAddress())
  })

  describe('isCollateralized', () => {
    it('should return true for OVM_Proposer', async () => {
      expect(
        await mockOVM_BondManager.isCollateralized(await sequencer.getAddress())
      ).to.equal(true)
    })

    it('should return false for non-sequencer', async () => {
      expect(
        await mockOVM_BondManager.isCollateralized(
          await nonSequencer.getAddress()
        )
      ).to.equal(false)
    })
  })
})
