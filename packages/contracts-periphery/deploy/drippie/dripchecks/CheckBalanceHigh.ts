/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('CheckBalanceHigh', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['CheckBalanceHigh']),
    from: deployer,
    log: true,
  })

  await deploy()
}

deployFn.tags = ['CheckBalanceHigh', 'DrippieEnvironment']

export default deployFn
