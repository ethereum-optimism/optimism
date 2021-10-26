/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { getContractFromArtifact, waitUntilTrue } from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  // There is a risk that on a fresh deployment we could get front-run,
  // and the Proxy would be bricked. But that feels unlikely, and we can recover from it.
  console.log(`Initializing Proxy__L1CrossDomainMessenger...`)
  const proxy = await getContractFromArtifact(
    hre,
    'Proxy__OVM_L1CrossDomainMessenger',
    {
      iface: 'L1CrossDomainMessenger',
      signerOrProvider: deployer,
    }
  )
  const libAddressManager = await getContractFromArtifact(hre, 'Lib_AddressManager')
  await proxy.initialize(libAddressManager.address)

  console.log(`Checking that contract was correctly initialized...`)
  await waitUntilTrue(async () => {
    return hexStringEquals(
      await proxy.libAddressManager(),
      libAddressManager.address
    )
  })
}

deployFn.tags = ['finalize']

export default deployFn
