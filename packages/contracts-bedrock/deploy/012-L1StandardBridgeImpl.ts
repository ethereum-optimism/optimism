import { DeployFunction } from 'hardhat-deploy/dist/types'
import { constants } from 'ethers'

import { predeploys } from '../scripts'
import { assertContractVariable, deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'L1StandardBridge',
    args: [],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'MESSENGER', constants.AddressZero)
      await assertContractVariable(
        contract,
        'OTHER_BRIDGE',
        predeploys.L2StandardBridge
      )
    },
  })
}

deployFn.tags = ['L1StandardBridgeImpl', 'setup', 'l1']

export default deployFn
