/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  getLiveContract,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  // There is a risk that on a fresh deployment we could get front-run,
  // and the Proxy would be bricked. But that feels unlikely, and we can recover from it.
  console.log(`Initializing Proxy__L1CrossDomainMessenger...`)
  const proxy = getLiveContract('Proxy__L1CrossDomainMessenger')
  await proxy.initialize(Lib_AddressManager.address)

  console.log(`Checking that contract was correctly initialized...`)
  await waitUntilTrue(async () => {
    return hexStringEquals(
      await proxy.libAddressManager(),
      Lib_AddressManager.address
    )
  })
}

deployFn.tags = ['fresh', 'finalize']

export default deployFn
