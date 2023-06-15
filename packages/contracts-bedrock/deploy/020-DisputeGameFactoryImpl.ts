import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  // We only want to deploy the dgf on devnet for now
  if (hre.deployConfig.l1ChainID === 900) {
    const finalOwner = hre.deployConfig.finalSystemOwner
    const disputeGameFactory = await deploy({
      hre,
      name: 'DisputeGameFactory',
      args: [finalOwner],
    })
    const fetchedOwner = await disputeGameFactory.owner()
    assert(fetchedOwner === finalOwner)
    console.log('DisputeGameFactory deployed at ' + disputeGameFactory.address)
  }
}

deployFn.tags = ['DisputeGameFactoryImpl', 'setup', 'l1']

export default deployFn
