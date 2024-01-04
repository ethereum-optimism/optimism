import { DeployFunction } from 'hardhat-deploy/dist/types'

import { predeploys } from '../scripts'
import {
  assertContractVariable,
  deploy,
  getContractFromArtifact,
} from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const L1CrossDomainMessengerProxy = await getContractFromArtifact(
    hre,
    'Proxy__OVM_L1CrossDomainMessenger'
  )
  await deploy({
    hre,
    name: 'L1ERC721Bridge',
    args: [L1CrossDomainMessengerProxy.address],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'MESSENGER',
        L1CrossDomainMessengerProxy.address
      )
      await assertContractVariable(
        contract,
        'OTHER_BRIDGE',
        predeploys.L2ERC721Bridge
      )
    },
  })
}

deployFn.tags = ['L1ERC721BridgeImpl', 'setup', 'l1']

export default deployFn
