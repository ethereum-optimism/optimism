/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

import { getDeployConfig } from '../../src'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const config = getDeployConfig(hre.network.name)

  const { deploy } = await hre.deployments.deterministic('Drippie', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Drippie']),
    from: deployer,
    args: [config.drippieOwner],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['Drippie']
deployFn.dependencies = ['OptimismAuthority']

export default deployFn
