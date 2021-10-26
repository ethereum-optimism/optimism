/* Imports: External */
import { hexStringEquals } from '@eth-optimism/core-utils'
import { ethers } from 'hardhat'
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getLiveContract, waitUntilTrue } from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const addressDictator = await getLiveContract(hre, 'AddressDictator', {
    signerOrProvider: deployer,
  })
  const libAddressManager = await getLiveContract(hre, 'Lib_AddressManager')
  const namedAddresses = await addressDictator.getNamedAddresses()
  const finalOwner = await addressDictator.finalOwner()
  let currentOwner = await libAddressManager.owner()

  console.log(
    '\n',
    'An Address Dictator contract has been deployed, with the following name/address pairs:'
  )
  for (const namedAddress of namedAddresses) {
    // Set alignment for readability
    const padding = ' '.repeat(40 - namedAddress.name.length)
    console.log(`${namedAddress.name}${padding}  ${namedAddress.addr}`)
  }
  console.log(
    '\n',
    'Please verify the values above, and the deployment steps up to this point,'
  )
  console.log(
    `  then transfer ownership of the Address Manager at (${libAddressManager.address})`
  )
  console.log(
    `  to the Address Dictator contract at ${addressDictator.address}.`
  )

  const hreSigners = await hre.ethers.getSigners()
  const hreSignerAddresses = hreSigners.map((signer) => {
    return signer.address
  })
  if (
    hreSignerAddresses.some((addr) => {
      return hexStringEquals(addr, currentOwner)
    })
  ) {
    console.log(
      'Deploy script owns the address manager, this must be CI. Setting addresses...'
    )
    const owner = await hre.ethers.getSigner(currentOwner)
    await libAddressManager
      .connect(owner)
      .transferOwnership(addressDictator.address)
  }

  await waitUntilTrue(
    async () => {
      console.log('Checking ownership of Lib_AddressManager... ')
      currentOwner = await libAddressManager.owner()
      console.log('Lib_AddressManager owner is now set to AddressDictator.')
      return hexStringEquals(currentOwner, addressDictator.address)
    },
    {
      // Try every 30 seconds for 500 minutes.
      delay: 30_000,
      retries: 1000,
    }
  )

  // Set the addresses!
  console.log('Ownership successfully transferred. Invoking setAddresses...')
  await addressDictator.setAddresses()

  currentOwner = await libAddressManager.owner()
  console.log('Verifying final ownership of Lib_AddressManager')
  if (!hexStringEquals(finalOwner, currentOwner)) {
    throw new Error(
      `The current address manager owner ${currentOwner}, \nis not equal to the expected owner: ${finalOwner}`
    )
  } else {
    console.log(`Address Manager ownership was returned to ${finalOwner}.`)
  }
}

deployFn.tags = ['fresh', 'upgrade', 'set-addresses']

export default deployFn
