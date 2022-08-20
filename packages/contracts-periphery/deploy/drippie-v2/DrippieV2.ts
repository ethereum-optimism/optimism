/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('DrippieV2', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['DrippieV2']),
    from: deployer,
    args: [hre.deployConfig.ddd],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['DrippieV2', 'DrippieEnvironmentV2']

export default deployFn
