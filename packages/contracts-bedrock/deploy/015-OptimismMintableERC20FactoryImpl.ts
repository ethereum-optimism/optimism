import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'OptimismMintableERC20Factory',
    args: [],
  })
}

deployFn.tags = ['OptimismMintableERC20FactoryImpl', 'setup', 'l1']

export default deployFn
