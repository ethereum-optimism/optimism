import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const admin = await hre.deployments.get('ProxyAdmin')
  const ProxyAdmin = await hre.ethers.getContractAt('ProxyAdmin', admin.address)

  // This is set up for fresh networks only
  const proxies = [
    'L2OutputOracleProxy',
    'L1CrossDomainMessengerProxy',
    'L1StandardBridgeProxy',
    'OptimismPortalProxy',
    'OptimismMintableERC20FactoryProxy',
    'L1ERC721BridgeProxy',
  ]

  // Wait on all the txs in parallel so that the deployment goes faster
  const txs = []
  for (const proxy of proxies) {
    const deployment = await hre.deployments.get(proxy)
    const Proxy = await hre.ethers.getContractAt('Proxy', deployment.address)
    const tx = await Proxy.changeAdmin(admin.address)
    txs.push(tx)
  }
  await Promise.all(txs.map((tx) => tx.wait()))

  const addressManager = await hre.deployments.get('AddressManager')
  const AddressManager = await hre.ethers.getContractAt(
    'AddressManager',
    addressManager.address
  )

  const postConfig = [
    await AddressManager.transferOwnership(admin.address),
    await ProxyAdmin.setAddressManager(addressManager.address),
  ]
  await Promise.all(postConfig.map((tx) => tx.wait()))
}

deployFn.tags = ['ConfigureProxyAdmin']

export default deployFn
