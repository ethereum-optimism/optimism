import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'

import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [proxyAdmin, addressManager] = await getContractsFromArtifacts(hre, [
    {
      name: 'ProxyAdmin',
      signerOrProvider: deployer,
    },
    {
      name: 'Lib_AddressManager',
      signerOrProvider: deployer,
    },
  ])

  let addressManagerOwner = await addressManager.callStatic.owner()
  if (addressManagerOwner !== proxyAdmin.address) {
    console.log(
      `AddressManager(${addressManager.address}).transferOwnership(${proxyAdmin.address})`
    )
    const tx = await addressManager.transferOwnership(proxyAdmin.address)
    await tx.wait()
  }

  addressManagerOwner = await addressManager.callStatic.owner()
  assert(
    addressManagerOwner === proxyAdmin.address,
    'AddressManager owner not set correctly'
  )
}

deployFn.tags = ['AddressManager', 'l1']

export default deployFn
