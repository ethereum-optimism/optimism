/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('Drippie', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Drippie']),
    from: deployer,
    args: [hre.deployConfig.ddd],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['Drippie', 'DrippieEnvironment']

export default deployFn
