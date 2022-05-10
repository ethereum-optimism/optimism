import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { calculateL1GasUsed, calculateL1Fee } from '@eth-optimism/core-utils'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import { deploy } from '../../../helpers'

describe('OVM_GasPriceOracle', () => {
  const initialGasPrice = 0

  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let OVM_GasPriceOracle: Contract
  beforeEach(async () => {
    OVM_GasPriceOracle = await deploy('OVM_GasPriceOracle', {
      signer: signer1,
      args: [signer1.address],
    })

    await OVM_GasPriceOracle.setOverhead(2750)
    await OVM_GasPriceOracle.setScalar(1500000)
    await OVM_GasPriceOracle.setDecimals(6)
  })

  describe('owner', () => {
    it('should have an owner', async () => {
      expect(await OVM_GasPriceOracle.owner()).to.equal(signer1.address)
    })
  })

  describe('setGasPrice', () => {
    it('should revert if called by someone other than the owner', async () => {
      await expect(OVM_GasPriceOracle.connect(signer2).setGasPrice(1234)).to.be
        .reverted
    })

    it('should succeed if called by the owner and is equal to `0`', async () => {
      await expect(OVM_GasPriceOracle.setGasPrice(0)).to.not.be.reverted
    })

    it('should emit event', async () => {
      await expect(OVM_GasPriceOracle.setGasPrice(100))
        .to.emit(OVM_GasPriceOracle, 'GasPriceUpdated')
        .withArgs(100)
    })
  })

  describe('get gasPrice', () => {
    it('should return zero at first', async () => {
      expect(await OVM_GasPriceOracle.gasPrice()).to.equal(initialGasPrice)
    })

    it('should change when setGasPrice is called', async () => {
      const gasPrice = 1234

      await OVM_GasPriceOracle.setGasPrice(gasPrice)

      expect(await OVM_GasPriceOracle.gasPrice()).to.equal(gasPrice)
    })

    it('is the 1st storage slot', async () => {
      await OVM_GasPriceOracle.setGasPrice(333433)

      expect(await OVM_GasPriceOracle.gasPrice()).to.equal(
        ethers.BigNumber.from(
          await signer1.provider.getStorageAt(OVM_GasPriceOracle.address, 1)
        )
      )
    })
  })

  describe('setL1BaseFee', () => {
    it('should revert if called by someone other than the owner', async () => {
      await expect(OVM_GasPriceOracle.connect(signer2).setL1BaseFee(1234)).to.be
        .reverted
    })

    it('should succeed if called by the owner', async () => {
      await expect(OVM_GasPriceOracle.setL1BaseFee(0)).to.not.be.reverted
    })

    it('should emit event', async () => {
      await expect(OVM_GasPriceOracle.setL1BaseFee(100))
        .to.emit(OVM_GasPriceOracle, 'L1BaseFeeUpdated')
        .withArgs(100)
    })
  })

  describe('get l1BaseFee', () => {
    it('should return zero at first', async () => {
      expect(await OVM_GasPriceOracle.l1BaseFee()).to.equal(initialGasPrice)
    })

    it('should change when setL1BaseFee is called', async () => {
      const baseFee = 1234
      await OVM_GasPriceOracle.setL1BaseFee(baseFee)
      expect(await OVM_GasPriceOracle.l1BaseFee()).to.equal(baseFee)
    })

    it('is the 2nd storage slot', async () => {
      await OVM_GasPriceOracle.setGasPrice(12345)

      expect(await OVM_GasPriceOracle.l1BaseFee()).to.equal(
        ethers.BigNumber.from(
          await signer1.provider.getStorageAt(OVM_GasPriceOracle.address, 2)
        )
      )
    })
  })

  // Test cases for gas estimation
  const inputs = [
    '0x',
    '0x00',
    '0x01',
    '0x0001',
    '0x0101',
    '0xffff',
    '0x00ff00ff00ff00ff00ff00ff',
  ]

  describe('getL1GasUsed', async () => {
    for (const input of inputs) {
      it(`case: ${input}`, async () => {
        const overhead = await OVM_GasPriceOracle.overhead()

        expect(await OVM_GasPriceOracle.getL1GasUsed(input)).to.deep.equal(
          calculateL1GasUsed(input, overhead)
        )
      })
    }
  })

  describe('getL1Fee', async () => {
    for (const input of inputs) {
      it(`case: ${input}`, async () => {
        await OVM_GasPriceOracle.setGasPrice(1)
        await OVM_GasPriceOracle.setL1BaseFee(1)

        expect(await OVM_GasPriceOracle.getL1Fee(input)).to.deep.equal(
          calculateL1Fee(
            input,
            await OVM_GasPriceOracle.overhead(),
            await OVM_GasPriceOracle.l1BaseFee(),
            await OVM_GasPriceOracle.scalar(),
            await OVM_GasPriceOracle.decimals()
          )
        )
      })
    }
  })

  describe('setOverhead', () => {
    it('should revert if called by someone other than the owner', async () => {
      await expect(OVM_GasPriceOracle.connect(signer2).setOverhead(1234)).to.be
        .reverted
    })

    it('should succeed if called by the owner', async () => {
      await expect(OVM_GasPriceOracle.setOverhead(0)).to.not.be.reverted
    })

    it('should emit event', async () => {
      await expect(OVM_GasPriceOracle.setOverhead(100))
        .to.emit(OVM_GasPriceOracle, 'OverheadUpdated')
        .withArgs(100)
    })
  })

  describe('get overhead', () => {
    it('should return 2750 at first', async () => {
      expect(await OVM_GasPriceOracle.overhead()).to.equal(2750)
    })

    it('should change when setOverhead is called', async () => {
      const overhead = 6657
      await OVM_GasPriceOracle.setOverhead(overhead)
      expect(await OVM_GasPriceOracle.overhead()).to.equal(overhead)
    })

    it('is the 3rd storage slot', async () => {
      await OVM_GasPriceOracle.setOverhead(119090)

      expect(await OVM_GasPriceOracle.overhead()).to.equal(
        ethers.BigNumber.from(
          await signer1.provider.getStorageAt(OVM_GasPriceOracle.address, 3)
        )
      )
    })
  })

  describe('setScalar', () => {
    it('should revert if called by someone other than the owner', async () => {
      await expect(OVM_GasPriceOracle.connect(signer2).setScalar(1234)).to.be
        .reverted
    })

    it('should succeed if called by the owner', async () => {
      await expect(OVM_GasPriceOracle.setScalar(0)).to.not.be.reverted
    })

    it('should emit event', async () => {
      await expect(OVM_GasPriceOracle.setScalar(100))
        .to.emit(OVM_GasPriceOracle, 'ScalarUpdated')
        .withArgs(100)
    })
  })

  describe('scalar', () => {
    it('should return 1 at first', async () => {
      expect(await OVM_GasPriceOracle.scalar()).to.equal(1500000)
    })

    it('should change when setScalar is called', async () => {
      const scalar = 9999
      await OVM_GasPriceOracle.setScalar(scalar)
      expect(await OVM_GasPriceOracle.scalar()).to.equal(scalar)
    })

    it('is the 4rd storage slot', async () => {
      await OVM_GasPriceOracle.setScalar(111111)

      expect(await OVM_GasPriceOracle.scalar()).to.equal(
        ethers.BigNumber.from(
          await signer1.provider.getStorageAt(OVM_GasPriceOracle.address, 4)
        )
      )
    })
  })

  describe('decimals', () => {
    it('is the 5th storage slot', async () => {
      expect(await OVM_GasPriceOracle.decimals()).to.equal(
        ethers.BigNumber.from(
          await signer1.provider.getStorageAt(OVM_GasPriceOracle.address, 5)
        )
      )
    })
  })
})
