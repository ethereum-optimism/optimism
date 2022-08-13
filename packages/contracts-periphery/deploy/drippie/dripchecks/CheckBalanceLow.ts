/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('CheckBalanceLow', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['CheckBalanceLow']),
    from: deployer,
    log: true,
  })

  await deploy()
}

deployFn.tags = ['CheckBalanceLow', 'DrippieEnvironment']

export default deployFn
