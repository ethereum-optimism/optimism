/* External Imports */
import { ethers } from 'hardhat'
import { ContractFactory, Contract, Signer } from 'ethers'

import { expect } from '../../../setup'

describe('OVM_ETH', () => {
  let signer1: Signer
  let signer2: Signer
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let Factory__OVM_ETH: ContractFactory
  before(async () => {
    Factory__OVM_ETH = await ethers.getContractFactory('OVM_ETH')
  })

  let OVM_ETH: Contract
  beforeEach(async () => {
    OVM_ETH = await Factory__OVM_ETH.deploy()
  })

  describe('transfer', () => {
    it('should revert', async () => {
      await expect(
        OVM_ETH.transfer(await signer2.getAddress(), 100)
      ).to.be.revertedWith(
        'OVM_ETH: transfer is disabled pending further community discussion.'
      )
    })
  })

  describe('approve', () => {
    it('should revert', async () => {
      await expect(
        OVM_ETH.approve(await signer2.getAddress(), 100)
      ).to.be.revertedWith(
        'OVM_ETH: approve is disabled pending further community discussion.'
      )
    })
  })

  describe('transferFrom', () => {
    it('should revert', async () => {
      await expect(
        OVM_ETH.transferFrom(
          await signer1.getAddress(),
          await signer2.getAddress(),
          100
        )
      ).to.be.revertedWith(
        'OVM_ETH: transferFrom is disabled pending further community discussion.'
      )
    })
  })

  describe('increaseAllowance', () => {
    it('should revert', async () => {
      await expect(
        OVM_ETH.increaseAllowance(await signer2.getAddress(), 100)
      ).to.be.revertedWith(
        'OVM_ETH: increaseAllowance is disabled pending further community discussion.'
      )
    })
  })

  describe('decreaseAllowance', () => {
    it('should revert', async () => {
      await expect(
        OVM_ETH.decreaseAllowance(await signer2.getAddress(), 100)
      ).to.be.revertedWith(
        'OVM_ETH: decreaseAllowance is disabled pending further community discussion.'
      )
    })
  })
})
