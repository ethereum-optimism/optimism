import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'ethers'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'
import { predeploys } from '../src/constants'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'OptimismMintableERC20Factory',
    args: [predeploys.L2StandardBridge],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'BRIDGE',
        ethers.utils.getAddress(predeploys.L2StandardBridge)
      )
    },
  })
}

deployFn.tags = ['OptimismMintableERC20FactoryImpl', 'l2']

export default deployFn
