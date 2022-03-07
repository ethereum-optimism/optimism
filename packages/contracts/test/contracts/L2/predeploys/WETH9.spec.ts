/* External Imports */
import { ethers } from 'hardhat'
import { Contract, Signer, ContractFactory } from 'ethers'
import {
  smock,
  MockContractFactory,
  MockContract,
} from '@defi-wonderland/smock'

/* Internal Imports */
import { expect } from '../../../setup'

describe('WETH9', () => {
  let signer: Signer
  let otherSigner: Signer
  let signerAddress: string
  let otherSignerAddress: string

  let Mock__Factory_WETH9: MockContractFactory<ContractFactory>
  let Mock__WETH9: MockContract<Contract>
  before(async () => {
    ;[signer, otherSigner] = await ethers.getSigners()
    signerAddress = await signer.getAddress()
    otherSignerAddress = await otherSigner.getAddress()
  })

  beforeEach(async () => {
    Mock__Factory_WETH9 = await smock.mock('WETH9')
    Mock__WETH9 = await Mock__Factory_WETH9.deploy()
  })

  describe('deposit', () => {
    it('should create WETH with fallback function', async () => {
      await expect(
        signer.sendTransaction({
          to: Mock__WETH9.address,
          value: 200,
        })
      ).to.not.be.reverted

      expect(await Mock__WETH9.balanceOf(signerAddress)).to.be.equal(200)
    })

    it('should create WETH with deposit function', async () => {
      await expect(Mock__WETH9.connect(signer).deposit({ value: 100 })).to.not
        .be.reverted

      expect(await Mock__WETH9.balanceOf(signerAddress)).to.be.equal(100)
    })
  })

  describe('withdraw', () => {
    it('should revert when withdraw amount is bigger than balance', async () => {
      await expect(Mock__WETH9.connect(signer).withdraw(10000)).to.be.reverted
    })

    it('should withdraw to eth', async () => {
      await Mock__WETH9.connect(signer).deposit({ value: 100 })
      await expect(Mock__WETH9.connect(signer).withdraw(50)).to.not.be.reverted
      expect(await Mock__WETH9.balanceOf(signerAddress)).to.be.equal(50)
    })
  })

  describe('totalSupply', () => {
    it('should return the totalSupply', async () => {
      await expect(Mock__WETH9.totalSupply()).to.not.be.reverted
    })
  })

  describe('transfer', () => {
    it('should revert when sending more than deposited', async () => {
      await Mock__WETH9.connect(signer).deposit({ value: 100 })
      await expect(
        Mock__WETH9.connect(signer).transfer(otherSignerAddress, 500)
      ).to.be.reverted
    })

    it('should transfer WETH to an other address', async () => {
      await Mock__WETH9.connect(signer).deposit({ value: 100 })
      await expect(Mock__WETH9.connect(signer).transfer(otherSignerAddress, 50))
        .to.not.be.reverted

      expect(await Mock__WETH9.balanceOf(signerAddress)).to.be.equal(50)

      expect(await Mock__WETH9.balanceOf(otherSignerAddress)).to.be.equal(50)
    })
  })

  describe('transferFrom', () => {
    it('should revert when there is no allowance', async () => {
      await Mock__WETH9.connect(signer).deposit({ value: 100 })
      await expect(
        Mock__WETH9.connect(otherSigner).transferFrom(
          signerAddress,
          otherSignerAddress,
          50
        )
      ).to.be.reverted
    })

    it('should transfer WETH to an other address when there is approvement', async () => {
      await Mock__WETH9.connect(signer).deposit({ value: 100 })
      await Mock__WETH9.connect(signer).approve(otherSignerAddress, 50)
      await expect(
        Mock__WETH9.connect(otherSigner).transferFrom(
          signerAddress,
          otherSignerAddress,
          50
        )
      ).to.not.be.reverted
    })
  })
})
