import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'ethers'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'L1Block',
    args: [],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'DEPOSITOR_ACCOUNT',
        ethers.utils.getAddress('0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001')
      )
    },
  })
}

deployFn.tags = ['L1BlockImpl', 'l2']

export default deployFn
