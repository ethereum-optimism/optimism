import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'L1FeeVault',
    args: [hre.deployConfig.l1FeeVaultRecipient],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'RECIPIENT',
        hre.deployConfig.l1FeeVaultRecipient
      )
    },
  })
}

deployFn.tags = ['L1FeeVaultImpl', 'l2']

export default deployFn
