import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [proxyAdmin, l1ERC721BridgeProxy, l1ERC721BridgeImpl] =
    await getContractsFromArtifacts(hre, [
      {
        name: 'ProxyAdmin',
        signerOrProvider: deployer,
      },
      {
        name: 'L1ERC721BridgeProxy',
        iface: 'L1ERC721Bridge',
        signerOrProvider: deployer,
      },
      {
        name: 'L1ERC721Bridge',
      },
    ])

  try {
    const tx = await proxyAdmin.upgrade(
      l1ERC721BridgeProxy.address,
      l1ERC721BridgeImpl.address
    )
    await tx.wait()
  } catch (e) {
    console.log('L1ERC721Bridge already initialized')
  }

  const version = await l1ERC721BridgeProxy.callStatic.version()
  console.log(`L1ERC721Bridge version: ${version}`)

  console.log('Upgraded L1ERC721Bridge')
}

deployFn.tags = ['L1ERC721BridgeInitialize', 'l1']

export default deployFn
