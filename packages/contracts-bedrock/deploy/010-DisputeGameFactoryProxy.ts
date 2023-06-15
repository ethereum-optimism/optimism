import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy, getDeploymentAddress } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const proxyAdmin = await getDeploymentAddress(hre, 'ProxyAdmin')

  // We only want to deploy the dgf on devnet for now
  if (hre.deployConfig.l1ChainID === 900) {
    console.log('Devnet detected, deploying DisputeGameFactoryProxy')
    const disputeGameFactoryProxy = await deploy({
      hre,
      name: 'DisputeGameFactoryProxy',
      contract: 'Proxy',
      args: [proxyAdmin],
    })
    console.log(
      'DisputeGameFactoryProxy deployed at ' + disputeGameFactoryProxy.address
    )
  }
}

deployFn.tags = ['DisputeGameFactoryProxy', 'setup', 'l1']

export default deployFn
