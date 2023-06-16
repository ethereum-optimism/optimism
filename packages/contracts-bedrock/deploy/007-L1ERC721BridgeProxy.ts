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
    name: 'L1ERC721BridgeProxy',
    contract: 'Proxy',
    args: [proxyAdmin],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'admin', proxyAdmin)
    },
  })
}

deployFn.tags = ['L1ERC721BridgeProxy', 'setup', 'l1']

export default deployFn
