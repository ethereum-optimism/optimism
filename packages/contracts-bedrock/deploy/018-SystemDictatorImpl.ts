import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { deployAndVerifyAndThen } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deployAndVerifyAndThen({
    hre,
    name: 'SystemDictator',
    args: [],
  })
}

deployFn.tags = ['SystemDictatorImpl']

export default deployFn
