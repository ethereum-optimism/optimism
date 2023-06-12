import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  getDeploymentAddress,
  deploy,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const proxyAdmin = await getDeploymentAddress(hre, 'ProxyAdmin')

  await deploy({
    hre,
    name: 'Proxy__OVM_L1StandardBridge',
    contract: 'L1ChugSplashProxy',
    args: [proxyAdmin],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'getOwner', proxyAdmin)
    },
  })
}

deployFn.tags = ['L1StandardBridgeProxy', 'setup', 'l1']

export default deployFn
