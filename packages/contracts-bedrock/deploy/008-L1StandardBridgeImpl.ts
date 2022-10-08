import { DeployFunction } from 'hardhat-deploy/dist/types'

import { predeploys } from '../src'
import {
  assertContractVariable,
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const L1CrossDomainMessengerProxy = await getContractFromArtifact(
    hre,
    'L1CrossDomainMessengerProxy'
  )
  await deployAndVerifyAndThen({
    hre,
    name: 'L1StandardBridgeImpl',
    contract: 'L1StandardBridge',
    args: [L1CrossDomainMessengerProxy.address],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'messenger',
        L1CrossDomainMessengerProxy.address
      )
      await assertContractVariable(
        contract,
        'otherBridge',
        predeploys.L2StandardBridge
      )
    },
  })
}

deployFn.tags = ['L1StandardBridgeImpl', 'fresh', 'migration']

export default deployFn
