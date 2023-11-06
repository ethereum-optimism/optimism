import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  deploy,
  getContractFromArtifact,
} from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const OptimismPortalProxy = await getContractFromArtifact(
    hre,
    'OptimismPortalProxy'
  )

  await deploy({
    hre,
    name: 'PortalSender',
    args: [OptimismPortalProxy.address],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'PORTAL',
        OptimismPortalProxy.address
      )
    },
  })
}

deployFn.tags = ['PortalSenderImpl', 'setup', 'l1']

export default deployFn
