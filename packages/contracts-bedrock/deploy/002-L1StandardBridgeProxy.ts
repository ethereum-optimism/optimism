import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  deployAndVerifyAndThen,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  await deployAndVerifyAndThen({
    hre,
    name: 'Proxy__OVM_L1StandardBridge',
    contract: 'L1ChugSplashProxy',
    args: [deployer],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'getOwner', deployer)
    },
  })
}

deployFn.tags = ['L1StandardBridgeProxy']

export default deployFn
