/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('Gelato', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Gelato']),
    args: ['0x527a819db1eb0e34426297b03bae11F2f8B3A19E'],
    from: deployer,
    log: true,
  })

  await deploy()
}

deployFn.tags = ['Gelato', 'DrippieEnvironmentV2']

export default deployFn
