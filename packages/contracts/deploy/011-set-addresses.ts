/* Imports: External */
import { hexStringEquals } from '@eth-optimism/core-utils'
import { ethers } from 'hardhat'
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  getContractFromArtifact,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  // We use this task to print out the list of addresses that will be updated by the
  // AddressDictator contract. The idea here is that the owner of the AddressManager will then
  // review these names and addresses before transferring ownership to the AddressDictator.
  // Once ownership has been transferred to the AddressDictator, we execute `setAddresses` which
  // triggers a series of setAddress calls on the AddressManager and then transfers ownership back
  // to the original owner.

  // First get relevant contract references.
  const AddressDictator = await getContractFromArtifact(
    hre,
    'AddressDictator',
    {
      signerOrProvider: deployer,
    }
  )
  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    'Lib_AddressManager'
  )
  const namedAddresses: Array<{ name: string; addr: string }> =
    await AddressDictator.getNamedAddresses()
  const finalOwner = await AddressDictator.finalOwner()
  const currentOwner = await Lib_AddressManager.owner()

  // Check if the hardhat runtime environment has the owner of the AddressManager. This will only
  // happen in CI. If this is the case, we can skip directly to transferring ownership over to the
  // AddressDictator contract.
  const hreSigners = await hre.ethers.getSigners()
  const hreHasOwner = hreSigners.some((signer) => {
    return hexStringEquals(signer.address, currentOwner)
  })

  if (hreHasOwner) {
    // Hardhat has the owner loaded into it, we can skip directly to transferOwnership.
    const owner = await hre.ethers.getSigner(currentOwner)
    await Lib_AddressManager.connect(owner).transferOwnership(
      AddressDictator.address
    )
  } else {
    console.log(`
      The AddressDictator contract (glory to Arstotzka) has been deployed.

      Name/Address pairs:
      ${namedAddresses.map((namedAddress) => {
        const padding = ' '.repeat(40 - namedAddress.name.length)
        return `
          ${namedAddress.name}${padding}  ${namedAddress.addr}
        `
      })}

      Current AddressManager owner: ${currentOwner}
      Final AddressManager owner: ${finalOwner}

      Please verify the values above, and the deployment steps up to this point,
        then transfer ownership of the AddressManager at ${
          Lib_AddressManager.address
        }
        to the AddressDictator contract at ${AddressDictator.address}.
    `)
  }

  // Wait for ownership to be transferred to the AddressDictator contract.
  await waitUntilTrue(
    async () => {
      return hexStringEquals(
        await Lib_AddressManager.owner(),
        AddressDictator.address
      )
    },
    {
      // Try every 30 seconds for 500 minutes.
      delay: 30_000,
      retries: 1000,
    }
  )

  // Set the addresses!
  console.log('Ownership successfully transferred. Invoking setAddresses...')
  await AddressDictator.setAddresses()

  // Make sure ownership has been correctly sent back to the original owner.
  console.log('Verifying final ownership of Lib_AddressManager...')
  await waitUntilTrue(async () => {
    return hexStringEquals(await Lib_AddressManager.owner(), finalOwner)
  })
}

deployFn.tags = ['set-addresses', 'upgrade']

export default deployFn
