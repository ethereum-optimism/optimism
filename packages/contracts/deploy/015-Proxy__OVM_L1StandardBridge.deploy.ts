/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getDeployedContract } from '../src/hardhat-deploy-ethers'
import { predeploys } from '../src/predeploys'
import { NON_ZERO_ADDRESS } from '../test/helpers/constants'
import { getContractFactory } from '../src/contract-defs'

import l1StandardBridgeJson from '../artifacts/contracts/optimistic-ethereum/OVM/bridge/tokens/OVM_L1StandardBridge.sol/OVM_L1StandardBridge.json'

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
    contract: 'L1ChugSplashProxy',
    from: deployer,
    args: [deployer],
    log: true,
  })

  if (!result.newlyDeployed) {
    return
  }

  // Create a contract object at the Proxy address with the proxy interface.
  const Proxy__WithChugSplashInterface = await getDeployedContract(
    hre,
    'Proxy__OVM_L1StandardBridge',
    {
      signerOrProvider: deployer,
      iface: 'L1ChugSplashProxy',
    }
  )

  // Create a contract object at the Proxy address with the brige implementation interface.
  const Proxy__WithBridgeInterface = await getDeployedContract(
    hre,
    'Proxy__OVM_L1StandardBridge',
    {
      signerOrProvider: deployer,
      iface: 'OVM_L1StandardBridge',
    }
  )

  // Set the implementation code
  const bridgeCode = l1StandardBridgeJson.deployedBytecode
  await Proxy__WithChugSplashInterface.setCode(bridgeCode)

  // Set slot 0 to the L1 Messenger Address
  const l1MessengerAddress = await Lib_AddressManager.getAddress(
    'Proxy__OVM_L1CrossDomainMessenger'
  )
  await Proxy__WithChugSplashInterface.setStorage(
    hre.ethers.constants.HashZero,
    hre.ethers.utils.hexZeroPad(l1MessengerAddress, 32)
  )
  // Verify that the slot was set correctly
  const l1MessengerStored = await Proxy__WithBridgeInterface.callStatic.messenger()
  console.log('l1MessengerStored:', l1MessengerStored)
  if (l1MessengerStored !== l1MessengerAddress) {
    throw new Error(
      'L1 messenger address was not correctly set, check the key value used in setStorage'
    )
  }

  // Set Slot 1 to the L2 Standard Bridge Address
  await Proxy__WithChugSplashInterface.setStorage(
    hre.ethers.utils.hexZeroPad('0x01', 32),
    hre.ethers.utils.hexZeroPad(predeploys.OVM_L2StandardBridge, 32)
  )
  // Verify that the slot was set correctly
  const l2TokenBridgeStored = await Proxy__WithBridgeInterface.callStatic.l2TokenBridge()
  console.log('l2TokenBridgeStored:', l2TokenBridgeStored)
  if (l2TokenBridgeStored !== predeploys.OVM_L2StandardBridge) {
    throw new Error(
      'L2 bridge address was not correctly set, check the key value used in setStorage'
    )
  }

  // transfer ownership to Address Manager owner
  const addressManagerOwner = Lib_AddressManager.callStatic.owner()
  await Proxy__WithChugSplashInterface.setOwner(addressManagerOwner)

  // Todo: remove this after adding chugsplash proxy
  await Lib_AddressManager.setAddress(
    'Proxy__OVM_L1StandardBridge',
    result.address
  )
}

deployFn.dependencies = ['Lib_AddressManager', 'OVM_L1StandardBridge']
deployFn.tags = ['Proxy__OVM_L1StandardBridge']

export default deployFn
