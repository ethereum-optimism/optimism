/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getDeployedContract } from '../src/hardhat-deploy-ethers'
import { predeploys } from '../src/predeploys'
import { NON_ZERO_ADDRESS } from '../test/helpers/constants'
import { getContractFactory } from '../src/contract-defs'

import l1StandardBridgeJson from '../artifacts/contracts/optimistic-ethereum/OVM/bridge/tokens/OVM_L1StandardBridge.sol/OVM_L1StandardBridge.json

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
  const addressManagerOwner = '0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A'

  const result = await deploy('Proxy__OVM_L1StandardBridge', {
    contract: 'L1ChugSplashProxy',
    from: deployer,
    args: [addressManagerOwner],
    log: true,
  })

  if (!result.newlyDeployed) {
    return
  }

  const Proxy__OVM_L1StandardBridge = await getDeployedContract(
    hre,
    'Proxy__OVM_L1StandardBridge',
    {
      signerOrProvider: deployer,
      iface: 'L1ChugSplashProxy',
    }
  )

  // Set the implementation code
  const bridgeCode = l1StandardBridgeJson.deployedBytecode
  await Proxy__OVM_L1StandardBridge.setCode(bridgeCode)

  // Set slot 0 to the L1 Messenger Address
  const l1MessengerAddress = await Lib_AddressManager.getAddress('Proxy__OVM_L1CrossDomainMessenger')
  await Proxy__OVM_L1StandardBridge.setStorage(
    hre.ethers.constants.HashZero,
    hre.ethers.utils.hexZeroPad(l1MessengerAddress, 32)
  )

  // Set Slot 1 to the L2 Standard Bridge Address
  await Proxy__OVM_L1StandardBridge.setStorage(
    hre.ethers.utils.hexZeroPad("0x01", 32),
    hre.ethers.utils.hexZeroPad(predeploys.OVM_L2StandardBridge, 32)
  )

  // Get and initialize the implementation to disable it.

  const bridgeImplAddress = Proxy__OVM_L1StandardBridge.connect(
    // connect to a provider to make an eth_call
    hre.ethers.getDefaultProvider()
  ).getImplementation()
  const bridgeImplementation = await getContractFactory(
    'OVM_L1StandardBridge',
    deployer
  ).attach(bridgeImplAddress)

  await bridgeImplementation.initialize(
    NON_ZERO_ADDRESS,
    constants.AddressZero
  )

  // Todo: remove this after adding chugsplash proxy
  await Lib_AddressManager.setAddress('Proxy__OVM_L1StandardBridge', result.address)
}

deployFn.dependencies = ['Lib_AddressManager', 'OVM_L1StandardBridge']
deployFn.tags = ['Proxy__OVM_L1StandardBridge']

export default deployFn
