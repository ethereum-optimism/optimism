import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { ContractFactory, Contract, Signer } from 'ethers'

describe('OVM_SequencerEntrypoint', () => {
  let signer1: Signer
  let signer2: Signer
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let Factory__OVM_GasPriceOracle: ContractFactory
  before(async () => {
    Factory__OVM_GasPriceOracle = await ethers.getContractFactory(
      'OVM_GasPriceOracle'
    )
  })

  let OVM_GasPriceOracle: Contract
  beforeEach(async () => {
    OVM_GasPriceOracle = await Factory__OVM_GasPriceOracle.deploy(
      await signer1.getAddress()
    )
  })

  describe('owner', () => {
    it('should have an owner', async () => {
      expect(await OVM_GasPriceOracle.owner()).to.equal(
        await signer1.getAddress()
      )
    })
  })

  describe('setGasPrice', () => {
    it('should revert if called by someone other than the owner', async () => {
      await expect(
        OVM_GasPriceOracle.connect(signer2).setGasPrice(1234)
      ).to.be.reverted
    })

    it('should succeed if called by the owner', async () => {
      await expect(
        OVM_GasPriceOracle.connect(signer1).setGasPrice(1234)
      ).to.not.be.reverted
    })
  })

  describe('getGasPrice', () => {
    it('should return zero at first', async () => {
      expect(await OVM_GasPriceOracle.getGasPrice()).to.equal(0)
    })

    it('should change when setGasPrice is called', async () => {
      const congestionPrice = 1234

      await OVM_GasPriceOracle.connect(signer1).setGasPrice(
        congestionPrice
      )

      expect(await OVM_GasPriceOracle.getGasPrice()).to.equal(
        congestionPrice
      )
    })
  })
})
