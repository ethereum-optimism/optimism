import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import { deploy } from '../../../helpers'

describe('BondManager', () => {
  let sequencer: SignerWithAddress
  let nonSequencer: SignerWithAddress
  before(async () => {
    ;[sequencer, nonSequencer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  let BondManager: Contract
  beforeEach(async () => {
    AddressManager = await deploy('Lib_AddressManager')

    BondManager = await deploy('BondManager', {
      args: [AddressManager.address],
    })

    AddressManager.setAddress('OVM_Proposer', sequencer.address)
  })

  describe('isCollateralized', () => {
    it('should return true for OVM_Proposer', async () => {
      expect(await BondManager.isCollateralized(sequencer.address)).to.equal(
        true
      )
    })

    it('should return false for non-sequencer', async () => {
      expect(await BondManager.isCollateralized(nonSequencer.address)).to.equal(
        false
      )
    })
  })
})
