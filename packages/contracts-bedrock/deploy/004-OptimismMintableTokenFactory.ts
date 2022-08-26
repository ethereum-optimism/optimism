/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  const bridge = await hre.deployments.get('L1StandardBridge')

  await deploy('OptimismMintableERC20Factory', {
    from: deployer,
    args: [bridge.address],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const factory = await hre.deployments.get('OptimismMintableERC20Factory')

  const OptimismMintableERC20Factory = await hre.ethers.getContractAt(
    'OptimismMintableERC20Factory',
    factory.address
  )

  if (bridge.address !== (await OptimismMintableERC20Factory.bridge())) {
    throw new Error('bridge misconfigured')
  }
}

deployFn.tags = ['OptimismMintableERC20Factory']

export default deployFn
