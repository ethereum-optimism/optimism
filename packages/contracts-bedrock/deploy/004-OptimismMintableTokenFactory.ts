/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  await deploy('OptimismMintableTokenFactoryProxy', {
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  await deploy('OptimismMintableTokenFactory', {
    from: deployer,
    args: [],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const factory = await hre.deployments.get('OptimismMintableTokenFactory')
  const bridge = await hre.deployments.get('L1StandardBridgeProxy')
  const proxy = await hre.deployments.get('OptimismMintableTokenFactoryProxy')
  const Proxy = await hre.ethers.getContractAt('Proxy', proxy.address)

  const OptimismMintableTokenFactory = await hre.ethers.getContractAt(
    'OptimismMintableTokenFactory',
    proxy.address
  )

  const upgradeTx = await Proxy.upgradeToAndCall(
    factory.address,
    OptimismMintableTokenFactory.interface.encodeFunctionData(
      'initialize(address)',
      [bridge.address]
    )
  )
  await upgradeTx.wait()

  if (bridge.address !== (await OptimismMintableTokenFactory.bridge())) {
    throw new Error('bridge misconfigured')
  }
}

deployFn.tags = ['OptimismMintableTokenFactory']

export default deployFn
