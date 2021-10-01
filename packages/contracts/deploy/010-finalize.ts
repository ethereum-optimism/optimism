/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { sleep } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { getDeployedContract } from '../src/hardhat-deploy-ethers'

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
  if (remoteOwner === owner) {
    console.log(
      `✓ Not changing owner of Lib_AddressManager because it's already correctly set`
    )
    return
  }

  console.log(`Transferring ownership of Lib_AddressManager to ${owner}...`)
  const tx = await Lib_AddressManager.transferOwnership(owner)
  await tx.wait()

  // Retry this a few times in case the transaction data lags for some reason.
  let retries = 0
  let newRemoteOwner = await Lib_AddressManager.owner()
  while (newRemoteOwner.toLowerCase() === remoteOwner.toLowerCase()) {
    retries++
    if (retries > 10) {
      throw new Error(
        `Failed to transfer ownership of Lib_AddressManager to ${owner}`
      )
    }
    await sleep(5000)
    newRemoteOwner = await Lib_AddressManager.owner()
  }

  if (newRemoteOwner.toLowerCase() !== owner.toLowerCase()) {
    throw new Error(
      `\n**FATAL ERROR. THIS SHOULD NEVER HAPPEN. CHECK YOUR DEPLOYMENT.**:\n` +
        `Could not transfer ownership of Lib_AddressManager.\n` +
        `Attempted to set owner of Lib_AddressManager to: ${owner}\n` +
        `Actual owner after transaction: ${newRemoteOwner}\n` +
        `This could indicate a compromised deployment.`
    )
  }

  console.log(`✓ Set owner of Lib_AddressManager to: ${owner}`)
}

deployFn.dependencies = ['Lib_AddressManager']
deployFn.tags = ['finalize']

export default deployFn
