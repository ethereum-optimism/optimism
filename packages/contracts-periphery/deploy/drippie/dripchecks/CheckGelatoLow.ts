/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('CheckGelatoLow', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['CheckGelatoLow']),
    from: deployer,
    log: true,
  })

  await deploy()
}

deployFn.tags = ['CheckGelatoLow', 'DrippieEnvironment']

export default deployFn
