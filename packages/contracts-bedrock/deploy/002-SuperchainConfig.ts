import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'SuperchainConfig',
    args: [],
  })
}

deployFn.tags = ['SuperchainConfig', 'setup', 'l1']

export default deployFn
