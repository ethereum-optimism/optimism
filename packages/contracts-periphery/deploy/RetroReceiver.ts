/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

import { getDeployConfig } from '../src'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const config = getDeployConfig(hre.network.name)

  const { deploy } = await hre.deployments.deterministic('AssetReceiver', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['RetroReceiver']),
    from: deployer,
    args: [config.retroReceiverOwner],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['RetroReceiver']

export default deployFn
