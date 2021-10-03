/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  getDeployedContract,
  deployAndRegister,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getDeployedContract(
    hre,
    'Lib_AddressManager'
  )

  await deployAndRegister({
    hre,
    name: 'Proxy__L1CrossDomainMessenger',
    contract: 'Lib_ResolvedDelegateProxy',
    iface: 'L1CrossDomainMessenger',
    args: [Lib_AddressManager.address, 'L1CrossDomainMessenger'],
    postDeployAction: async (contract) => {
      console.log(`Initializing Proxy__L1CrossDomainMessenger...`)
      await contract.initialize(Lib_AddressManager.address)

      console.log(`Checking that contract was correctly initialized...`)
      await waitUntilTrue(async () => {
        return hexStringEquals(
          await contract.libAddressManager(),
          Lib_AddressManager.address
        )
      })
    },
  })
}

deployFn.dependencies = ['Lib_AddressManager', 'L1CrossDomainMessenger']
deployFn.tags = ['Proxy__L1CrossDomainMessenger']

export default deployFn
