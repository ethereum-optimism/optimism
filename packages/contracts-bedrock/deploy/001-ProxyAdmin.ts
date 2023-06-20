import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  getContractsFromArtifacts,
  deploy,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [addressManager] = await getContractsFromArtifacts(hre, [
    {
      name: 'Lib_AddressManager',
      signerOrProvider: deployer,
    },
  ])

  const proxyAdmin = await deploy({
    hre,
    name: 'ProxyAdmin',
    args: [deployer],
    postDeployAction: async (contract) => {
      // Owner is temporarily set to the deployer.
      await assertContractVariable(contract, 'owner', deployer)
    },
  })

  let addressManagerOnProxy = await proxyAdmin.callStatic.addressManager()
  if (addressManagerOnProxy !== addressManager.address) {
    // Set the address manager on the proxy admin
    console.log(
      `ProxyAdmin(${proxyAdmin.address}).setAddressManager(${addressManager.address})`
    )
    const tx = await proxyAdmin.setAddressManager(addressManager.address)
    await tx.wait()
  }

  // Validate the address manager was set correctly.
  addressManagerOnProxy = await proxyAdmin.callStatic.addressManager()
  assert(
    addressManagerOnProxy === addressManager.address,
    'AddressManager not set on ProxyAdmin'
  )
}

deployFn.tags = ['ProxyAdmin', 'setup', 'l1']

export default deployFn
