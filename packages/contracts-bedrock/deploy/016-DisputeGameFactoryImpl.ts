import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  // We only want to deploy the dgf on devnet for now
  const network = await hre.ethers.provider.getNetwork()
  const chainId = network.chainId

  if (chainId === 900) {
    const disputeGameFactory = await deploy({
      hre,
      name: 'DisputeGameFactory',
      args: [],
    })
    console.log(`DisputeGameFactory deployed at ${disputeGameFactory.address}`)
  } else {
    console.log('Skipping deployment of DisputeGameFactory implementation')
  }
}

deployFn.tags = ['DisputeGameFactoryImpl', 'setup', 'l1']

export default deployFn
