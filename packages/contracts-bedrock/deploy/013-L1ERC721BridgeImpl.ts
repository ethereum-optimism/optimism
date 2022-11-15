import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'

import { predeploys } from '../src'
import {
  assertContractVariable,
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  let L1CrossDomainMessengerProxy: ethers.Contract
  try {
    L1CrossDomainMessengerProxy = await getContractFromArtifact(
      hre,
      'Proxy__OVM_L1CrossDomainMessenger'
    )
  } catch {
    L1CrossDomainMessengerProxy = await getContractFromArtifact(
      hre,
      'L1CrossDomainMessengerProxy'
    )
  }

  await deployAndVerifyAndThen({
    hre,
    name: 'L1ERC721Bridge',
    args: [L1CrossDomainMessengerProxy.address, predeploys.L2ERC721Bridge],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'MESSENGER',
        L1CrossDomainMessengerProxy.address
      )
    },
  })
}

deployFn.tags = ['L1ERC721BridgeImpl', 'fresh', 'migration']

export default deployFn
