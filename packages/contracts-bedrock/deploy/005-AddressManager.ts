import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@eth-optimism/hardhat-deploy-config'

// TODO(tynes): This needs to be deployed for fresh networks
// but not for upgrading existing networks
const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  await deploy('AddressManager', {
    from: deployer,
    args: [],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })
}

deployFn.tags = ['Lib_AddressManager', 'legacy']

export default deployFn
