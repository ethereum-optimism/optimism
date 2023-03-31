import { DeployFunction } from 'hardhat-deploy/dist/types'

import { predeploys } from '../src'
import {
  assertContractVariable,
  deploy,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const L1CrossDomainMessengerProxy = await getContractFromArtifact(
    hre,
    'Proxy__OVM_L1CrossDomainMessenger'
  )

  await deploy({
    hre,
    name: 'L1StandardBridge',
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
        predeploys.L2StandardBridge
      )
    },
  })
}

deployFn.tags = ['L1StandardBridgeImpl', 'setup', 'l1']

export default deployFn
