import { DeployFunction } from 'hardhat-deploy/dist/types'
import { constants } from 'ethers'

import { assertContractVariable, deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'OptimismMintableERC20Factory',
    args: [],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'BRIDGE', constants.AddressZero)
    },
  })
}

deployFn.tags = ['OptimismMintableERC20FactoryImpl', 'setup', 'l1']

export default deployFn
