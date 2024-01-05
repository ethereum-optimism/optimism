import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  deploy,
  getContractFromArtifact,
} from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const optimismPortalProxy = await getContractFromArtifact(
    hre,
    'OptimismPortalProxy'
  )
  await deploy({
    hre,
    name: 'L1CrossDomainMessenger',
    args: [optimismPortalProxy.address],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'PORTAL',
        optimismPortalProxy.address
      )
    },
  })
}

deployFn.tags = ['L1CrossDomainMessengerImpl', 'setup', 'l1']

export default deployFn
