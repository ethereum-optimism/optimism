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

  const result = await deploy('Proxy__OVM_L1StandardBridge', {
    contract: 'Lib_ResolvedDelegateProxy',
    from: deployer,
    args: [Lib_AddressManager.address, 'OVM_L1StandardBridge'],
    log: true,
  })

  if (!result.newlyDeployed) {
    return
  }

  const l1MessengerAddress = await Lib_AddressManager.getAddress('Proxy__OVM_L1CrossDomainMessenger')

  const Proxy__OVM_L1StandardBridge = await getDeployedContract(
    hre,
    'Proxy__OVM_L1StandardBridge',
    {
      signerOrProvider: deployer,
      iface: 'OVM_L1StandardBridge',
    }
  )

  await Proxy__OVM_L1StandardBridge.initialize(
    l1MessengerAddress,
    predeploys.OVM_L2StandardBridge
  )

  let messenger = await Proxy__OVM_L1StandardBridge.messenger()
  if(messenger !== l1MessengerAddress) {
    throw new Error(
      'Proxy__OVM_L1StandardBridge failed to initialize'
    )
  }
  // Todo: remove this after adding chugsplash proxy
  await Lib_AddressManager.setAddress('Proxy__OVM_L1StandardBridge', result.address)
}

deployFn.dependencies = ['Lib_AddressManager', 'OVM_L1StandardBridge']
deployFn.tags = ['Proxy__OVM_L1StandardBridge']

export default deployFn
