import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy, getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [addressManager] = await getContractsFromArtifacts(hre, [
    {
      name: 'Lib_AddressManager',
      signerOrProvider: deployer,
    },
  ])

  // The name in the address manager for this contract
  const name = 'OVM_L1CrossDomainMessenger'

  console.log(
    `Setting up ResolvedDelegateProxy with AddressManager(${addressManager.address})`
  )

  const l1CrossDomainMessengerProxy = await deploy({
    hre,
    name: 'Proxy__OVM_L1CrossDomainMessenger',
    contract: 'ResolvedDelegateProxy',
    args: [addressManager.address, name],
  })

  let addr = await addressManager.getAddress(name)
  if (addr !== l1CrossDomainMessengerProxy.address) {
    console.log(
      `AddressManager(${addressManager.address}).setAddress(${name}, ${l1CrossDomainMessengerProxy.address})`
    )
    const tx = await addressManager.setAddress(
      name,
      l1CrossDomainMessengerProxy.address
    )
    await tx.wait()
  }

  addr = await addressManager.getAddress(name)
  assert(
    addr === l1CrossDomainMessengerProxy.address,
    `${name} not set correctly`
  )
}

deployFn.tags = ['L1CrossDomainMessengerProxy', 'setup', 'l1']

export default deployFn
