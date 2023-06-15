<<<<<<< HEAD
<<<<<<< HEAD
import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  // We only want to deploy the dgf on devnet for now
  if (hre.deployConfig.l1ChainID === 900) {
    const disputeGameFactory = await deploy({
      hre,
      name: 'DisputeGameFactory',
      args: [],
    })
    console.log('DisputeGameFactory deployed at ' + disputeGameFactory.address)
  }
}

deployFn.tags = ['DisputeGameFactoryImpl', 'setup', 'l1']

export default deployFn
||||||| parent of b998a6a67 (DisputeGameFactory devnet deploy scripts)
=======
import assert from 'assert'

||||||| parent of 4c1b5e2ac (Remove final owner from dgf impl initialization.)
import assert from 'assert'

=======
>>>>>>> 4c1b5e2ac (Remove final owner from dgf impl initialization.)
import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  // We only want to deploy the dgf on devnet for now
  if (hre.deployConfig.l1ChainID === 900) {
    const disputeGameFactory = await deploy({
      hre,
      name: 'DisputeGameFactory',
      args: [],
    })
    console.log('DisputeGameFactory deployed at ' + disputeGameFactory.address)
  }
}

deployFn.tags = ['DisputeGameFactoryImpl', 'setup', 'l1']

export default deployFn
>>>>>>> b998a6a67 (DisputeGameFactory devnet deploy scripts)
