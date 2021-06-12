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

  const OVM_L1CrossDomainMessenger = await getDeployedContract(hre, 'Proxy__OVM_L1CrossDomainMessenger', {
    signerOrProvider: deployer,
  })

  const result = await deploy('MVM_L1MetisGateway', {
    contract: 'OVM_L1ERC20Gateway',
    from: deployer,
    args: [],
    log: true,
  })

  if (!result.newlyDeployed) {
    return
  }

  await Lib_AddressManager.setAddress('MVM_L1MetisGateway', result.address)
}

deployFn.dependencies = ['Lib_AddressManager']
deployFn.tags = ['MVM_L1MetisGateway']

export default deployFn
