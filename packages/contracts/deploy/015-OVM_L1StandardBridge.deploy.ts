/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { constants } from 'ethers'

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

  const result = await deploy('OVM_L1StandardBridge', {
    from: deployer,
    args: [],
    log: true,
  })

  if (!result.newlyDeployed) {
    return
  }

  const OVM_L1StandardBridge = await getDeployedContract(hre, 'OVM_L1StandardBridge', {
    signerOrProvider: deployer,
  })

  const l1MessengerAddress = await Lib_AddressManager.getAddress('Proxy__OVM_L1CrossDomainMessenger')

  // NOTE: this initialization is *not* technically required (we only need to initialize the proxy)
  // but it feels safer to initialize this anyway. Otherwise someone else could come along and
  // initialize this. We'll set all the addresses to 0 so that this copy is unusable.
  await OVM_L1StandardBridge.initialize(
    constants.AddressZero,
    constants.AddressZero,
    constants.AddressZero
  )

  // Todo: remove this after adding chugsplash proxy
  await Lib_AddressManager.setAddress('OVM_L1StandardBridge', result.address)
}

deployFn.dependencies = ['Lib_AddressManager']
deployFn.tags = ['OVM_L1StandardBridge']

export default deployFn
