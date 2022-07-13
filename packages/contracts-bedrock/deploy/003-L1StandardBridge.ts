/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  await deploy('L1StandardBridgeProxy', {
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const messenger = await hre.deployments.get('L1CrossDomainMessengerProxy')

  await deploy('L1StandardBridge', {
    from: deployer,
    args: [messenger.address],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const proxy = await hre.deployments.get('L1StandardBridgeProxy')
  const Proxy = await hre.ethers.getContractAt('Proxy', proxy.address)
  const bridge = await hre.deployments.get('L1StandardBridge')

  const L1StandardBridge = await hre.ethers.getContractAt(
    'L1StandardBridge',
    proxy.address
  )

  const upgradeTx = await Proxy.upgradeToAndCall(
    bridge.address,
    L1StandardBridge.interface.encodeFunctionData('initialize(address)', [
      messenger.address,
    ])
  )
  await upgradeTx.wait()

  if (messenger.address !== (await L1StandardBridge.messenger())) {
    throw new Error('misconfigured messenger')
  }
}

deployFn.tags = ['L1StandardBridge']

export default deployFn
