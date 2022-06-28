/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  // This needs to be deployed for the indexer
  await deploy('Lib_AddressManager', {
    from: deployer,
    args: [],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })
}

deployFn.tags = ['Lib_AddressManager', 'legacy']

export default deployFn
