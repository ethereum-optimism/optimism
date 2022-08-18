/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('CheckTrue', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['CheckTrue']),
    from: deployer,
    log: true,
  })

  await deploy()
}

deployFn.tags = ['CheckTrue', 'DrippieEnvironment']

export default deployFn
