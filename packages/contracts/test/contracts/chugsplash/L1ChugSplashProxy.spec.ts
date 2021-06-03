import { expect } from '../../setup'

/* Imports: External */
import hre, { ethers } from 'hardhat'
import { Contract, Signer } from 'ethers'

describe('L1ChugSplashProxy', () => {
  let signer1: Signer
  let signer2: Signer
  before(async () => {
    ;[signer1, signer2] = await hre.ethers.getSigners()
  })

  let L1ChugSplashProxy: Contract
  beforeEach(async () => {
    const Factory__L1ChugSplashProxy = await hre.ethers.getContractFactory(
      'L1ChugSplashProxy'
    )
    L1ChugSplashProxy = await Factory__L1ChugSplashProxy.deploy(
      await signer1.getAddress()
    )
  })

  describe('getOwner', () => {
    it('should return the owner if called by the owner', async () => {
      expect(
        await L1ChugSplashProxy.connect(signer1).callStatic.getOwner()
      ).to.equal(await signer1.getAddress())
    })

    it('should return the owner if called by the zero address in an eth_call', async () => {
      expect(
        await L1ChugSplashProxy.connect(signer1.provider).callStatic.getOwner({
          from: hre.ethers.constants.AddressZero,
        })
      ).to.equal(await signer1.getAddress())
    })

    it('should otherwise pass through to the proxied contract', async () => {
      await expect(
        L1ChugSplashProxy.connect(signer2).callStatic.getOwner()
      ).to.be.revertedWith('L1ChugSplashProxy: implementation is not set yet')
    })
  })

  describe('setOwner', () => {
    it('should succeed if called by the owner', async () => {
      await expect(
        L1ChugSplashProxy.connect(signer1).setOwner(await signer2.getAddress())
      ).to.not.be.reverted

      expect(
        await L1ChugSplashProxy.connect(signer2).callStatic.getOwner()
      ).to.equal(await signer2.getAddress())
    })

    it('should otherwise pass through to the proxied contract', async () => {
      await expect(
        L1ChugSplashProxy.connect(signer2).setOwner(await signer1.getAddress())
      ).to.be.revertedWith('L1ChugSplashProxy: implementation is not set yet')
    })
  })

  describe('setStorage', () => {})
})
