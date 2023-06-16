import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy, getDeploymentAddress } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const proxyAdmin = await getDeploymentAddress(hre, 'ProxyAdmin')

  // We only want to deploy the dgf on devnet for now
  const network = await hre.ethers.provider.getNetwork()
  const chainId = network.chainId

  if (chainId === 900) {
    console.log('Devnet detected, deploying DisputeGameFactoryProxy')
    const disputeGameFactoryProxy = await deploy({
      hre,
      name: 'DisputeGameFactoryProxy',
      contract: 'Proxy',
      args: [proxyAdmin],
    })
    console.log(
      `DisputeGameFactoryProxy deployed at ${disputeGameFactoryProxy.address}`
    )
  } else {
    console.log('Skipping deployment of DisputeGameFactoryProxy')
  }
}

deployFn.tags = ['DisputeGameFactoryProxy', 'setup', 'l1']

export default deployFn
