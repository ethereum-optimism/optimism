/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { predeploys } from '@eth-optimism/contracts'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  await hre.deployments.deploy('L2ERC721Bridge', {
    from: deployer,
    args: [predeploys.L2CrossDomainMessenger],
    log: true,
  })
}

deployFn.tags = ['L2ERC721Bridge']

export default deployFn
