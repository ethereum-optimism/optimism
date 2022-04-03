/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'

import { names } from '../src/address-names'
import { getDeployConfig } from '../src/deploy-config'

/* Imports: External */

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const deployConfig = getDeployConfig(hre.network.name)

  await deploy(names.unmanaged.Lib_AddressManager, {
    from: deployer,
    args: [],
    log: true,
    waitConfirmations: deployConfig.numDeployConfirmations,
  })
}

// This is kept during an upgrade. So no upgrade tag.
deployFn.tags = ['Lib_AddressManager']

export default deployFn
