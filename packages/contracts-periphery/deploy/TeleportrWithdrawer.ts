/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic(
    'TeleportrWithdrawer',
    {
      salt: hre.ethers.utils.solidityKeccak256(
        ['string'],
        ['TeleportrWithdrawer']
      ),
      from: deployer,
      args: [hre.deployConfig.ddd],
      log: true,
    }
  )

  await deploy()
}

deployFn.tags = ['TeleportrWithdrawer']

export default deployFn
