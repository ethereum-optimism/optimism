import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const L1StandardBridgeProxy = await getContractFromArtifact(
    hre,
    'Proxy__OVM_L1StandardBridge'
  )

  await deployAndVerifyAndThen({
    hre,
    name: 'OptimismMintableERC20Factory',
    args: [L1StandardBridgeProxy.address],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'BRIDGE',
        L1StandardBridgeProxy.address
      )
    },
  })
}

deployFn.tags = ['OptimismMintableERC20FactoryImpl']

export default deployFn
