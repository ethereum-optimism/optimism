import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'

import { deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'SystemConfig',
    args: [],
  })
}

deployFn.tags = ['SystemConfigImpl', 'setup', 'l1']

export default deployFn
