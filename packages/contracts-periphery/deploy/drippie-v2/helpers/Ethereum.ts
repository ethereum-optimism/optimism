/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('Ethereum', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Ethereum']),
    from: deployer,
    log: true,
  })

  await deploy()
}

deployFn.tags = ['Ethereum', 'DrippieEnvironmentV2']

export default deployFn
