import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const L1StandardBridgeProxy = await getContractFromArtifact(
    hre,
    'L1StandardBridgeProxy'
  )
  await deployAndVerifyAndThen({
    hre,
    name: 'OptimismMintableERC20FactoryImpl',
    contract: 'OptimismMintableERC20Factory',
    args: [L1StandardBridgeProxy.address],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'bridge',
        L1StandardBridgeProxy.address
      )
    },
  })
}

deployFn.tags = ['OptimismMintableERC20FactoryImpl', 'fresh', 'migration']

export default deployFn
