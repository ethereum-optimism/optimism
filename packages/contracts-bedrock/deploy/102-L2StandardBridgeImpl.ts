import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'ethers'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const Artifact__L1StandardBridge = await hre.companionNetworks[
    'l1'
  ].deployments.get('L1StandardBridgeProxy')

  await deploy({
    hre,
    name: 'L2StandardBridge',
    args: [Artifact__L1StandardBridge.address],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'OTHER_BRIDGE',
        ethers.utils.getAddress(Artifact__L1StandardBridge.address)
      )
    },
  })
}

deployFn.tags = ['L2StandardBridgeImpl', 'l2']

export default deployFn
