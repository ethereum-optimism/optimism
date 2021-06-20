/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getDeployedContract } from '../src/hardhat-deploy-ethers'
import { predeploys } from '../src/predeploys'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  const Lib_AddressManager = await getDeployedContract(
    hre,
    'Lib_AddressManager',
    {
      signerOrProvider: deployer,
    }
  )

  const result = await deploy('Proxy__MVM_AddressManager', {
    contract: 'Lib_ResolvedDelegateProxy',
    from: deployer,
    args: [Lib_AddressManager.address, 'MVM_AddressManager'],
    log: true,
  })

  if (!result.newlyDeployed) {
    return
  }

  const Proxy__MVM_AddressManager = await getDeployedContract(
    hre,
    'Proxy__MVM_AddressManager',
    {
      signerOrProvider: deployer,
      iface: 'MVM_AddressManager',
    }
  )

  await Lib_AddressManager.setAddress('Proxy__MVM_AddressManager', result.address)
}

deployFn.dependencies = ['Lib_AddressManager', 'MVM_AddressManager']
deployFn.tags = ['Proxy__MVM_AddressManager']

export default deployFn
