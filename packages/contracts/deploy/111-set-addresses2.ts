/* Imports: External */
import { hexStringEquals } from '@eth-optimism/core-utils'
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  getDeployedContract,
  getReusableContract,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

// todo: reduce redundancy between this and 071
const deployFn: DeployFunction = async (hre) => {
  const addressSetter2 = await getDeployedContract(hre, 'AddressSetter2')
  const libAddressManager = await getReusableContract(hre, 'Lib_AddressManager')
  const names = await addressSetter2.getNames()
  const addresses = await addressSetter2.getAddresses()
  const finalOwner = await addressSetter2.finalOwner()

  console.log(
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
  console.log(`  to the Address Setter contract at ${addressSetter2.address}.`)

  await waitUntilTrue(
    async () => {
      console.log('Checking ownership of Lib_AddressManager')
      return hexStringEquals(
        await libAddressManager.owner(),
        addressSetter2.address
      )
    },
    {
      // Try every 30 seconds for 500 minutes.
      delay: 30_000,
      retries: 1000,
    }
  )

  // Set the addresses!
  await addressSetter2.setAddresses()

  const currentOwner = await libAddressManager.owner()
  console.log('Verifying final ownership of Lib_AddressManager')
  if (!hexStringEquals(finalOwner, currentOwner)) {
    // todo: pause here get user input deciding whether to continue or exit?
    console.log(
      `The current address manager owner ${currentOwner}, \nis not equal to the expected owner: ${finalOwner}`
    )
  }
}

deployFn.tags = ['set-addresses', 'upgrade']

export default deployFn
