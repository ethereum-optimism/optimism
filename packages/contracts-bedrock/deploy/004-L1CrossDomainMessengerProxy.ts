import { DeployFunction } from 'hardhat-deploy/dist/types'

import { deploy, getDeploymentAddress } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const addressManager = await getDeploymentAddress(hre, 'Lib_AddressManager')

  await deploy({
    hre,
    name: 'Proxy__OVM_L1CrossDomainMessenger',
    contract: 'ResolvedDelegateProxy',
    args: [addressManager, 'OVM_L1CrossDomainMessenger'],
  })
}

deployFn.tags = ['L1CrossDomainMessengerProxy', 'setup', 'l1']

export default deployFn
