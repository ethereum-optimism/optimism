/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  await deploy('Lib_AddressManager', {
    from: deployer,
    args: [],
    log: true,
    waitConfirmations: (hre as any).deployConfig.numDeployConfirmations,
  })
}

deployFn.tags = ['Lib_AddressManager']

export default deployFn
