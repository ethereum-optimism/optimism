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

  // Deploy the Proxy
  const resultProxyDeploy = await deploy('Proxy__L1StandardBridge', {
    contract: 'L1ChugSplashProxy',
    from: deployer,
    args: [deployer],
    log: true,
  })

  if (!resultProxyDeploy.newlyDeployed) {
    return
  }

  // Deploy the L1 standard bridge implementation
  const l1MessengerAddress = await Lib_AddressManager.getAddress(
    'Proxy__L1CrossDomainMessenger'
  )

  const resultL1BridgeDeploy = await deploy('L1StandardBridge', {
    contract: 'L1StandardBridge',
    from: deployer,
    args: [l1MessengerAddress, predeploys.L2StandardBridge],
    log: true,
  })

  if (!resultL1BridgeDeploy.newlyDeployed) {
    return
  }

  // Create a contract object at the Proxy address with the proxy interface.
  const Proxy__WithChugSplashInterface = await getDeployedContract(
    hre,
    'Proxy__L1StandardBridge',
    {
      signerOrProvider: deployer,
      iface: 'L1ChugSplashProxy',
    }
  )

  // Create a contract object at the Proxy address with the brige implementation interface.
  const Proxy__WithBridgeInterface = await getDeployedContract(
    hre,
    'Proxy__L1StandardBridge',
    {
      signerOrProvider: deployer,
      iface: 'L1StandardBridge',
    }
  )

  // Set the implementation code
  const bridgeCode = await hre.ethers.provider.getCode(resultL1BridgeDeploy.address)
  await Proxy__WithChugSplashInterface.setCode(bridgeCode)

  // Clear Slot 0 which used to hold the L1 Messenger Address
  await Proxy__WithChugSplashInterface.setStorage(
    hre.ethers.constants.HashZero,
    hre.ethers.utils.hexZeroPad(hre.ethers.constants.AddressZero, 32)
  )
  // Clear Slot 1 which used to hold the L2 Standard Bridge Address
  await Proxy__WithChugSplashInterface.setStorage(
    hre.ethers.utils.hexZeroPad('0x01', 32),
    hre.ethers.utils.hexZeroPad(hre.ethers.constants.AddressZero, 32)
  )

  // Verify the immutable messenger property is correct
  const l1MessengerStored =
    await Proxy__WithBridgeInterface.callStatic.messenger()

  if (l1MessengerStored !== l1MessengerAddress) {
    throw new Error(
      'L1 messenger address was not correctly set, check the key value used in setStorage'
    )
  }

  // Verify the immutable l2TokenBridge property is correct
  const l2TokenBridgeStored =
    await Proxy__WithBridgeInterface.callStatic.l2TokenBridge()

  if (l2TokenBridgeStored !== predeploys.L2StandardBridge) {
    throw new Error(
      'L2 bridge address was not correctly set, check the key value used in setStorage'
    )
  }

  // transfer ownership to Address Manager owner
  const addressManagerOwner = Lib_AddressManager.callStatic.owner()
  await Proxy__WithChugSplashInterface.setOwner(addressManagerOwner)

  // Todo: remove this after adding chugsplash proxy
  await Lib_AddressManager.setAddress('Proxy__L1StandardBridge', resultProxyDeploy.address)
}

deployFn.dependencies = ['Lib_AddressManager', 'L1StandardBridge']
deployFn.tags = ['Proxy__L1StandardBridge']

export default deployFn
