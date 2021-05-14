/* Imports: External */
import hre from 'hardhat'
import { expect } from 'chai'

/* Imports: Internal */
import { smockit, isMockContract } from '../../src'

describe('[smock]: initialization tests', () => {
  const ethers = (hre as any).ethers

  describe('initialization: ethers objects', () => {
    it('should be able to create a SmockContract from an ethers ContractFactory', async () => {
      const spec = await ethers.getContractFactory('TestHelpers_EmptyContract')
      const mock = await smockit(spec)

      expect(isMockContract(mock)).to.be.true
    })

    it('should be able to create a SmockContract from an ethers Contract', async () => {
      const factory = await ethers.getContractFactory(
        'TestHelpers_EmptyContract'
      )

      const spec = await factory.deploy()
      const mock = await smockit(spec)

      expect(isMockContract(mock)).to.be.true
    })

    it('should be able to create a SmockContract from an ethers Interface', async () => {
      const factory = await ethers.getContractFactory(
        'TestHelpers_EmptyContract'
      )

      const spec = factory.interface
      const mock = await smockit(spec)

      expect(isMockContract(mock)).to.be.true
    })
  })

  describe('initialization: other', () => {
    it('should be able to create a SmockContract from a contract name', async () => {
      const spec = 'TestHelpers_EmptyContract'
      const mock = await smockit(spec)

      expect(isMockContract(mock)).to.be.true
    })

    it('should be able to create a SmockContract from a JSON contract artifact object', async () => {
      const artifact = await hre.artifacts.readArtifact(
        'TestHelpers_BasicReturnContract'
      )
      const spec = artifact
      const mock = await smockit(spec)

      expect(isMockContract(mock)).to.be.true
    })

    it('should be able to create a SmockContract from a JSON contract ABI object', async () => {
      const artifact = await hre.artifacts.readArtifact(
        'TestHelpers_BasicReturnContract'
      )
      const spec = artifact.abi
      const mock = await smockit(spec)

      expect(isMockContract(mock)).to.be.true
    })

    it('should be able to create a SmockContract from a JSON contract ABI string', async () => {
      const artifact = await hre.artifacts.readArtifact(
        'TestHelpers_BasicReturnContract'
      )
      const spec = JSON.stringify(artifact.abi)
      const mock = await smockit(spec)

      expect(isMockContract(mock)).to.be.true
    })
  })
})
