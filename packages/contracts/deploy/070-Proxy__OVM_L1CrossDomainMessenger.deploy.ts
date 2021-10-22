/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getReusableContract,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getReusableContract(
    hre,
    'Lib_AddressManager'
  )

  // todo: this fails when trying to do a fresh deploy, because Lib_ResolvedDelegateProxy
  // requires that the implementation has already been set in the Address Manager.
  // The revert message is: 'Target address must be initialized'
  await deployAndPostDeploy({
    hre,
    name: 'Proxy__OVM_L1CrossDomainMessenger',
    contract: 'Lib_ResolvedDelegateProxy',
    iface: 'L1CrossDomainMessenger',
    args: [Lib_AddressManager.address, 'OVM_L1CrossDomainMessenger'],
    // This reverts on a fresh deploy, because the implementation is not yet added to the AddressManager.
    // I think the best option is to do the initialization atomically from within the AddressSetter.
    // postDeployAction: async (contract) => {
    //   console.log(`Initializing Proxy__OVM_L1CrossDomainMessenger...`)
    //   await contract.initialize(Lib_AddressManager.address)

    //   console.log(`Checking that contract was correctly initialized...`)
    //   await waitUntilTrue(async () => {
    //     return hexStringEquals(
    //       await contract.libAddressManager(),
    //       Lib_AddressManager.address
    //     )
    //   })
    // },
  })
}

// This is kept during an upgrade. So no upgrade tag.
deployFn.tags = ['fresh', 'Proxy__OVM_L1CrossDomainMessenger']

export default deployFn
