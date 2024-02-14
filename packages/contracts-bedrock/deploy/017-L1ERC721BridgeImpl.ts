import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'L1ERC721Bridge',
    args: [],
  })
}

deployFn.tags = ['L1ERC721BridgeImpl', 'setup', 'l1']

export default deployFn
