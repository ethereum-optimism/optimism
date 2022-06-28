/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  await deploy('ProxyAdmin', {
    from: deployer,
    args: [deployer],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const admin = await hre.deployments.get('ProxyAdmin')

  // TODO(tynes): this will need to be modified for mainnet
  // as the API for the proxies will be different
  const proxies = [
    'L2OutputOracleProxy',
    'L1CrossDomainMessengerProxy',
    'L1StandardBridgeProxy',
    'OptimismPortalProxy',
    'OptimismMintableTokenFactoryProxy',
  ]
  for (const proxy of proxies) {
    const deployment = await hre.deployments.get(proxy)
    const Proxy = await hre.ethers.getContractAt('Proxy', deployment.address)
    const tx = await Proxy.changeAdmin(admin.address)
    await tx.wait()
  }
}

deployFn.tags = ['ProxyAdmin']

export default deployFn
