/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('AssetReceiver', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['RetroReceiver']),
    from: deployer,
    args: [hre.deployConfig.ddd],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['RetroReceiver']

export default deployFn
