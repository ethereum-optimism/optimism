import { DeployFunction } from 'hardhat-deploy/dist/types'
import { constants } from 'ethers'

import { predeploys } from '../scripts'
import { assertContractVariable, deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'L1ERC721Bridge',
    args: [],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'MESSENGER', constants.AddressZero)
      await assertContractVariable(
        contract,
        'OTHER_BRIDGE',
        predeploys.L2ERC721Bridge
      )
    },
  })
}

deployFn.tags = ['L1ERC721BridgeImpl', 'setup', 'l1']

export default deployFn
