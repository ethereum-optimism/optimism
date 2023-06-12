import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [proxyAdmin, optimismPortalProxy, optimismPortalImpl] =
    await getContractsFromArtifacts(hre, [
      {
        name: 'ProxyAdmin',
        signerOrProvider: deployer,
      },
      {
        name: 'OptimismPortalProxy',
        iface: 'OptimismPortal',
        signerOrProvider: deployer,
      },
      {
        name: 'OptimismPortal',
      },
    ])

  // Initialize the portal, setting paused to false
  try {
    const tx = await proxyAdmin.upgradeAndCall(
      optimismPortalProxy.address,
      optimismPortalImpl.address,
      optimismPortalProxy.interface.encodeFunctionData('initialize', [false])
    )
    await tx.wait()
  } catch (e) {
    console.log('OptimismPortal already initialized')
  }

  const isPaused = await optimismPortalProxy.callStatic.paused()
  assert(isPaused === false)

  console.log('Upgraded and initialized OptimismPortal')
  const version = await optimismPortalProxy.callStatic.version()
  console.log(`OptimismPortal version: ${version}`)
}

deployFn.tags = ['OptimismPortalInitialize', 'l1']

export default deployFn
