/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('Math', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Math']),
    from: deployer,
    log: true,
  })

  await deploy()
}

deployFn.tags = ['Math', 'DrippieEnvironmentV2']

export default deployFn
