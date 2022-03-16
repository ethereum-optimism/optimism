import hre from 'hardhat'
import { Contract, Signer } from 'ethers'
import { smock } from '@defi-wonderland/smock'

import { expect } from '../../setup'
import { getContractInterface } from '../../../src'
import { deploy } from '../../helpers'

describe('L1ChugSplashProxy', () => {
  let signer1: Signer
  let signer2: Signer
  before(async () => {
    ;[signer1, signer2] = await hre.ethers.getSigners()
  })

  let L1ChugSplashProxy: Contract
  beforeEach(async () => {
    L1ChugSplashProxy = await deploy('L1ChugSplashProxy', {
      args: [await signer1.getAddress()],
    })
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

  describe('getImplementation', () => {
    it('should succeed if called by the owner', async () => {
      expect(
        await L1ChugSplashProxy.connect(signer1).callStatic.getImplementation()
      ).to.equal(hre.ethers.constants.AddressZero)
    })

    it('should succeed if called by the zero address in an eth_call', async () => {
      expect(
        await L1ChugSplashProxy.connect(
          hre.ethers.provider
        ).callStatic.getImplementation({
          from: hre.ethers.constants.AddressZero,
        })
      ).to.equal(hre.ethers.constants.AddressZero)
    })

    it('should otherwise pass through to the proxied contract', async () => {
      await expect(
        L1ChugSplashProxy.connect(signer2).getImplementation()
      ).to.be.revertedWith('L1ChugSplashProxy: implementation is not set yet')
    })
  })

  describe('setStorage', () => {
    it('should succeed if called by the owner', async () => {
      const storageKey = hre.ethers.utils.keccak256('0x1234')
      const storageValue = hre.ethers.utils.keccak256('0x5678')

      await expect(
        L1ChugSplashProxy.connect(signer1).setStorage(storageKey, storageValue)
      ).to.not.be.reverted

      expect(
        await hre.ethers.provider.getStorageAt(
          L1ChugSplashProxy.address,
          storageKey
        )
      ).to.equal(storageValue)
    })

    it('should otherwise pass through to the proxied contract', async () => {
      const storageKey = hre.ethers.utils.keccak256('0x1234')
      const storageValue = hre.ethers.utils.keccak256('0x5678')

      await expect(
        L1ChugSplashProxy.connect(signer2).setStorage(storageKey, storageValue)
      ).to.be.revertedWith('L1ChugSplashProxy: implementation is not set yet')
    })
  })

  describe('setCode', () => {
    it('should succeed if called by the owner', async () => {
      const code = '0x1234'

      await expect(L1ChugSplashProxy.connect(signer1).setCode(code)).to.not.be
        .reverted

      const implementation = await L1ChugSplashProxy.connect(
        signer1
      ).callStatic.getImplementation()

      expect(await hre.ethers.provider.getCode(implementation)).to.equal(code)
    })

    it('should not change the implementation address if the code does not change', async () => {
      const code = '0x1234'

      await L1ChugSplashProxy.connect(signer1).setCode(code)

      const implementation = await L1ChugSplashProxy.connect(
        signer1
      ).callStatic.getImplementation()

      await L1ChugSplashProxy.connect(signer1).setCode(code)

      expect(
        await L1ChugSplashProxy.connect(signer1).callStatic.getImplementation()
      ).to.equal(implementation)
    })
  })

  describe('fallback', () => {
    it('should revert if implementation is not set', async () => {
      await expect(
        signer1.sendTransaction({
          to: L1ChugSplashProxy.address,
          data: '0x',
        })
      ).to.be.revertedWith('L1ChugSplashProxy: implementation is not set yet')
    })

    it('should execute the proxied contract when the implementation is set', async () => {
      const code = '0x00' // STOP

      await L1ChugSplashProxy.connect(signer1).setCode(code)

      await expect(
        signer1.sendTransaction({
          to: L1ChugSplashProxy.address,
          data: '0x',
        })
      ).to.not.be.reverted
    })

    it('should throw an error if the owner has signalled an upgrade', async () => {
      const owner = await smock.fake<Contract>(
        getContractInterface('iL1ChugSplashDeployer')
      )

      L1ChugSplashProxy = await deploy('L1ChugSplashProxy', {
        args: [owner.address],
      })

      owner.isUpgrading.returns(true)

      await expect(
        owner.wallet.sendTransaction({
          to: L1ChugSplashProxy.address,
          data: '0x',
        })
      ).to.be.revertedWith(
        'L1ChugSplashProxy: system is currently being upgraded'
      )
    })
  })
})
