import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'SequencerFeeVault',
    args: [hre.deployConfig.sequencerFeeVaultRecipient],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'RECIPIENT',
        hre.deployConfig.sequencerFeeVaultRecipient
      )
    },
  })
}

deployFn.tags = ['SequencerFeeVaultImpl', 'l2']

export default deployFn
