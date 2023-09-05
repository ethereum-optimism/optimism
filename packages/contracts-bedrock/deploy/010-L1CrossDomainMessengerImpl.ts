import { DeployFunction } from 'hardhat-deploy/dist/types'
import { constants } from 'ethers'

import { assertContractVariable, deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'L1CrossDomainMessenger',
    args: [],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'PORTAL', constants.AddressZero)
    },
  })
}

deployFn.tags = ['L1CrossDomainMessengerImpl', 'setup', 'l1']

export default deployFn
