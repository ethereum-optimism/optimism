import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'BaseFeeVault',
    args: [hre.deployConfig.baseFeeVaultRecipient],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'RECIPIENT',
        hre.deployConfig.baseFeeVaultRecipient
      )
    },
  })
}

deployFn.tags = ['BaseFeeVaultImpl', 'l2']

export default deployFn
