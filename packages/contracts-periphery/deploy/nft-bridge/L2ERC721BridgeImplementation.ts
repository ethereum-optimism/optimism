/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'hardhat'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  await hre.deployments.deploy('L2ERC721Bridge', {
    from: deployer,
    args: [ethers.constants.AddressZero, ethers.constants.AddressZero],
    log: true,
  })
}

deployFn.tags = ['L2ERC721BridgeImplementation']
deployFn.dependencies = ['L2ERC721BridgeProxy']

export default deployFn
