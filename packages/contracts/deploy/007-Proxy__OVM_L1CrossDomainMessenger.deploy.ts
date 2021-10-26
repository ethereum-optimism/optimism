/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getContractFromArtifact,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getContractFromArtifact(
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
  })
}

// This is kept during an upgrade. So no upgrade tag.
deployFn.tags = ['Proxy__OVM_L1CrossDomainMessenger']

export default deployFn
