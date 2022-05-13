/* External Imports */
import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import { deploy } from '../../../helpers'

describe('OVM_ETH', () => {
  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let OVM_ETH: Contract
  beforeEach(async () => {
    OVM_ETH = await deploy('OVM_ETH')
  })

  describe('transfer', () => {
    it('should revert', async () => {
      await expect(OVM_ETH.transfer(signer2.address, 100)).to.be.revertedWith(
        'OVM_ETH: transfer is disabled pending further community discussion.'
      )
    })
  })

  describe('approve', () => {
    it('should revert', async () => {
      await expect(OVM_ETH.approve(signer2.address, 100)).to.be.revertedWith(
        'OVM_ETH: approve is disabled pending further community discussion.'
      )
    })
  })

  describe('transferFrom', () => {
    it('should revert', async () => {
      await expect(
        OVM_ETH.transferFrom(signer1.address, signer2.address, 100)
      ).to.be.revertedWith(
        'OVM_ETH: transferFrom is disabled pending further community discussion.'
      )
    })
  })

  describe('increaseAllowance', () => {
    it('should revert', async () => {
      await expect(
        OVM_ETH.increaseAllowance(signer2.address, 100)
      ).to.be.revertedWith(
        'OVM_ETH: increaseAllowance is disabled pending further community discussion.'
      )
    })
  })

  describe('decreaseAllowance', () => {
    it('should revert', async () => {
      await expect(
        OVM_ETH.decreaseAllowance(signer2.address, 100)
      ).to.be.revertedWith(
        'OVM_ETH: decreaseAllowance is disabled pending further community discussion.'
      )
    })
  })
})
