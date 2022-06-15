/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'hardhat'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  await hre.deployments.deploy('OptimismMintableERC721Factory', {
    from: deployer,
    args: [ethers.constants.AddressZero],
    log: true,
  })
}

deployFn.tags = ['OptimismMintableERC721FactoryImplementation']
deployFn.dependencies = ['OptimismMintableERC721FactoryProxy']

export default deployFn
