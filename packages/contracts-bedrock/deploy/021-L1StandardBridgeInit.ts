import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [proxyAdmin, l1StandardBridgeProxy, l1StandardBridgeImpl] =
    await getContractsFromArtifacts(hre, [
      {
        name: 'ProxyAdmin',
        signerOrProvider: deployer,
      },
      {
        name: 'Proxy__OVM_L1StandardBridge',
        iface: 'L1StandardBridge',
        signerOrProvider: deployer,
      },
      {
        name: 'L1StandardBridge',
      },
    ])

  const proxyType = await proxyAdmin.callStatic.proxyType(
    l1StandardBridgeProxy.address
  )
  if (proxyType !== 1) {
    console.log(
      `ProxyAdmin(${proxyAdmin.address}).setProxyType(${l1StandardBridgeProxy.address}, 1)`
    )
    // Set the L1StandardBridge to the UPGRADEABLE proxy type.
    const tx = await proxyAdmin.setProxyType(l1StandardBridgeProxy.address, 1)
    await tx.wait()
  }

  try {
    const tx = await proxyAdmin.upgrade(
      l1StandardBridgeProxy.address,
      l1StandardBridgeImpl.address
    )
    await tx.wait()
  } catch (e) {
    console.log('L1StandardBridge already initialized')
  }

  const version = await l1StandardBridgeProxy.callStatic.version()
  console.log(`L1StandardBridge version: ${version}`)

  console.log('Upgraded L1StandardBridge')
}

deployFn.tags = ['L1StandardBridgeInitialize', 'l1']

export default deployFn
