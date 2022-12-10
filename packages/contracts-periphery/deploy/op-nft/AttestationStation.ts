/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic('AttestationStation', {
    salt: hre.ethers.utils.solidityKeccak256(
      ['string'],
      ['AttestationStation']
    ),
    from: deployer,
    args: [hre.deployConfig.ddd],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['AttestationStation', 'AttestationStationEnvironment']

export default deployFn
