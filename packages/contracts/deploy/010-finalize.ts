/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  getDeployedContract,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const Lib_AddressManager = await getDeployedContract(
    hre,
    'Lib_AddressManager',
    {
      signerOrProvider: deployer,
    }
  )

  const owner = (hre as any).deployConfig.ovmAddressManagerOwner
  const remoteOwner = await Lib_AddressManager.owner()
  if (hexStringEquals(owner, remoteOwner)) {
    console.log(
      `✓ Not changing owner of Lib_AddressManager because it's already correctly set`
    )
    return
  }

  console.log(`Transferring ownership of Lib_AddressManager to ${owner}...`)
  await Lib_AddressManager.transferOwnership(owner)

  console.log(`Confirming transfer was successful...`)
  await waitUntilTrue(async () => {
    return hexStringEquals(await Lib_AddressManager.owner(), owner)
  })

  console.log(`✓ Set owner of Lib_AddressManager to: ${owner}`)
}

deployFn.dependencies = ['Lib_AddressManager']
deployFn.tags = ['finalize']

export default deployFn
