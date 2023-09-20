import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  if (process.env.ENABLE_BOBA_TOKEN_DEPLOYMENT === 'true') {
    await deploy({
      hre,
      name: 'BOBA',
      args: [],
    })
  } else {
    console.log('BOBA Token deployment is disabled')
  }
}

deployFn.tags = ['BOBA', 'Token', 'l1']

export default deployFn
