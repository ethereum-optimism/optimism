import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { smock, MockContract } from '@defi-wonderland/smock'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'

describe('WETH9', () => {
  let signer: SignerWithAddress
  let otherSigner: SignerWithAddress

  before(async () => {
    ;[signer, otherSigner] = await ethers.getSigners()
  })

  let Mock__WETH9: MockContract<Contract>
  beforeEach(async () => {
    Mock__WETH9 = await (await smock.mock('WETH9')).deploy()
  })

  describe('deposit', () => {
    it('should create WETH with fallback function', async () => {
      await expect(
        signer.sendTransaction({
          to: Mock__WETH9.address,
          value: 200,
        })
      ).to.not.be.reverted

      expect(await Mock__WETH9.balanceOf(signer.address)).to.be.equal(200)
    })

    it('should create WETH with deposit function', async () => {
      await expect(Mock__WETH9.deposit({ value: 100 })).to.not.be.reverted

      expect(await Mock__WETH9.balanceOf(signer.address)).to.be.equal(100)
    })
  })

  describe('withdraw', () => {
    it('should revert when withdraw amount is bigger than balance', async () => {
      await expect(Mock__WETH9.withdraw(10000)).to.be.reverted
    })

    it('should withdraw to eth', async () => {
      await Mock__WETH9.deposit({ value: 100 })
      await expect(Mock__WETH9.withdraw(50)).to.not.be.reverted
      expect(await Mock__WETH9.balanceOf(signer.address)).to.be.equal(50)
    })
  })

  describe('totalSupply', () => {
    it('should return the totalSupply', async () => {
      await expect(Mock__WETH9.totalSupply()).to.not.be.reverted
    })
  })

  describe('transfer', () => {
    it('should revert when sending more than deposited', async () => {
      await Mock__WETH9.deposit({ value: 100 })
      await expect(Mock__WETH9.transfer(otherSigner.address, 500)).to.be
        .reverted
    })

    it('should transfer WETH to an other address', async () => {
      await Mock__WETH9.deposit({ value: 100 })
      await expect(Mock__WETH9.transfer(otherSigner.address, 50)).to.not.be
        .reverted

      expect(await Mock__WETH9.balanceOf(signer.address)).to.be.equal(50)

      expect(await Mock__WETH9.balanceOf(otherSigner.address)).to.be.equal(50)
    })
  })

  describe('transferFrom', () => {
    it('should revert when there is no allowance', async () => {
      await Mock__WETH9.deposit({ value: 100 })
      await expect(
        Mock__WETH9.connect(otherSigner).transferFrom(
          signer.address,
          otherSigner.address,
          50
        )
      ).to.be.reverted
    })

    it('should transfer WETH to an other address when there is approvement', async () => {
      await Mock__WETH9.deposit({ value: 100 })
      await Mock__WETH9.approve(otherSigner.address, 50)
      await expect(
        Mock__WETH9.connect(otherSigner).transferFrom(
          signer.address,
          otherSigner.address,
          50
        )
      ).to.not.be.reverted
    })
  })
})
