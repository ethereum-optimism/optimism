/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  const messenger = await hre.deployments.get('L1CrossDomainMessenger')

  await deploy('L1StandardBridge', {
    from: deployer,
    args: [messenger.address],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const bridge = await hre.deployments.get('L1StandardBridge')

  const L1StandardBridge = await hre.ethers.getContractAt(
    'L1StandardBridge',
    bridge.address
  )

  if (messenger.address !== (await L1StandardBridge.messenger())) {
    throw new Error('misconfigured messenger')
  }
}

deployFn.tags = ['L1StandardBridge']

export default deployFn
