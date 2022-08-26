/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  await deploy('OptimismMintableERC20FactoryProxy', {
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const bridge = await hre.deployments.get('L1StandardBridgeProxy')

  await deploy('OptimismMintableERC20Factory', {
    from: deployer,
    args: [bridge.address],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const factory = await hre.deployments.get('OptimismMintableERC20Factory')
  const proxy = await hre.deployments.get('OptimismMintableERC20FactoryProxy')
  const Proxy = await hre.ethers.getContractAt('Proxy', proxy.address)

  const OptimismMintableERC20Factory = await hre.ethers.getContractAt(
    'OptimismMintableERC20Factory',
    proxy.address
  )

  const upgradeTx = await Proxy.upgradeTo(factory.address)
  await upgradeTx.wait()

  if (bridge.address !== (await OptimismMintableERC20Factory.bridge())) {
    throw new Error('bridge misconfigured')
  }
}

deployFn.tags = ['OptimismMintableERC20Factory']

export default deployFn
