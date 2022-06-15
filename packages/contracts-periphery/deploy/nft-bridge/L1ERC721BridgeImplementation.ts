/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'hardhat'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  await hre.deployments.deploy('L1ERC721Bridge', {
    from: deployer,
    args: [ethers.constants.AddressZero, ethers.constants.AddressZero],
    log: true,
  })
}

deployFn.tags = ['L1ERC721BridgeImplementation']
deployFn.dependencies = ['L1ERC721BridgeProxy']

export default deployFn
