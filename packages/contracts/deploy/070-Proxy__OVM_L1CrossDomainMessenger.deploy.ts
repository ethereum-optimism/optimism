/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getLiveContract,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getLiveContract(hre, 'Lib_AddressManager')

  await deployAndPostDeploy({
    hre,
    name: 'Proxy__OVM_L1CrossDomainMessenger',
    contract: 'Lib_ResolvedDelegateProxy',
    iface: 'L1CrossDomainMessenger',
    args: [Lib_AddressManager.address, 'OVM_L1CrossDomainMessenger'],
  })
}

// This is kept during an upgrade. So no upgrade tag.
deployFn.tags = ['fresh', 'Proxy__OVM_L1CrossDomainMessenger']

export default deployFn
