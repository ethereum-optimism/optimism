import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [proxyAdmin, l1CrossDomainMessengerProxy, l1CrossDomainMessengerImpl] =
    await getContractsFromArtifacts(hre, [
      {
        name: 'ProxyAdmin',
        signerOrProvider: deployer,
      },
      {
        name: 'Proxy__OVM_L1CrossDomainMessenger',
        iface: 'L1CrossDomainMessenger',
        signerOrProvider: deployer,
      },
      {
        name: 'L1CrossDomainMessenger',
      },
    ])

  const proxyType = await proxyAdmin.callStatic.proxyType(
    l1CrossDomainMessengerProxy.address
  )
  if (proxyType !== 2) {
    console.log(
      `ProxyAdmin(${proxyAdmin.address}).setProxyType(${l1CrossDomainMessengerProxy.address}, 2)`
    )
    // Set the L1CrossDomainMessenger to the RESOLVED proxy type.
    const tx = await proxyAdmin.setProxyType(
      l1CrossDomainMessengerProxy.address,
      2
    )
    await tx.wait()
  }

  const name = 'OVM_L1CrossDomainMessenger'

  const implementationName = proxyAdmin.implementationName(
    l1CrossDomainMessengerImpl.address
  )
  if (implementationName !== name) {
    console.log(
      `ProxyAdmin(${proxyAdmin.address}).setImplementationName(${l1CrossDomainMessengerImpl.address}, 'OVM_L1CrossDomainMessenger')`
    )
    const tx = await proxyAdmin.setImplementationName(
      l1CrossDomainMessengerProxy.address,
      name
    )
    await tx.wait()
  }

  try {
    const tx = await proxyAdmin.upgradeAndCall(
      l1CrossDomainMessengerProxy.address,
      l1CrossDomainMessengerImpl.address,
      l1CrossDomainMessengerImpl.interface.encodeFunctionData('initialize')
    )
    await tx.wait()
  } catch (e) {
    console.log('L1CrossDomainMessenger already initialized')
  }

  const version = await l1CrossDomainMessengerProxy.callStatic.version()
  console.log(`L1CrossDomainMessenger version: ${version}`)

  console.log('Upgraded L1CrossDomainMessenger')
}

deployFn.tags = ['L1CrossDomainMessengerInitialize', 'l1']

export default deployFn
