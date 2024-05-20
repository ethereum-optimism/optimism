import { DeployFunction } from 'hardhat-deploy/dist/types'

import { assertContractVariable, deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'SuperchainConfig',
    args: [],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'guardian',
        '0x0000000000000000000000000000000000000000'
      )
    },
  })
}

deployFn.tags = ['SuperchainConfigImpl', 'setup', 'l1']

export default deployFn
