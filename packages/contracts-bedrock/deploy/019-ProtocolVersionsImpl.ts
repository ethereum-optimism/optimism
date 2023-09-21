import { DeployFunction } from 'hardhat-deploy/dist/types'

import { assertContractVariable, deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'ProtocolVersions',
    args: [],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'owner',
        '0x000000000000000000000000000000000000dEaD'
      )
    },
  })
}

deployFn.tags = ['ProtocolVersionsImpl', 'setup', 'l1']

export default deployFn
