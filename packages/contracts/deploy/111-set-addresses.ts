/* Imports: External */
import { hexStringEquals } from '@eth-optimism/core-utils'
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getLiveContract, waitUntilTrue } from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const addressSetter = await getLiveContract(hre, 'AddressSetter')
  const libAddressManager = await getLiveContract(hre, 'Lib_AddressManager')
  const names = await addressSetter.getNames()
  const addresses = await addressSetter.getAddresses()
  const finalOwner = await addressSetter.finalOwner()
  let currentOwner

  console.log(
    '\n',
    'An Address Setter contract has been deployed, with the following address <=> name pairs:'
  )
  for (let i = 0; i < names.length; i++) {
    console.log(`${addresses[i]} <=>  ${names[i]}`)
  }
  console.log(
    '\n',
    'Please verify the values above, and the deployment steps up to this point,'
  )
  console.log(
    `  then transfer ownership of the Address Manager at (${libAddressManager.address})`
  )
  console.log(`  to the Address Setter contract at ${addressSetter.address}.`)

  await waitUntilTrue(
    async () => {
      currentOwner = await libAddressManager.owner()
      console.log('Checking ownership of Lib_AddressManager... ')
      return hexStringEquals(currentOwner, addressSetter.address)
    },
    {
      // Try every 30 seconds for 500 minutes.
      delay: 30_000,
      retries: 1000,
    }
  )

  // Set the addresses!
  await addressSetter.setAddresses()

  currentOwner = await libAddressManager.owner()
  console.log('Verifying final ownership of Lib_AddressManager')
  if (!hexStringEquals(finalOwner, currentOwner)) {
    console.log(
      `The current address manager owner ${currentOwner}, \nis not equal to the expected owner: ${finalOwner}`
    )
  }
}

deployFn.tags = ['fresh', 'upgrade', 'set-addresses']

export default deployFn
