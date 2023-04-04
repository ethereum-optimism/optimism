import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import { ethers } from 'ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const l1 = hre.network.companionNetworks['l1']
  const deployConfig = hre.getDeployConfig(l1)

  const sequencerFeeVaultRecipient = deployConfig.sequencerFeeVaultRecipient
  if (sequencerFeeVaultRecipient === ethers.constants.AddressZero) {
    throw new Error(`SequencerFeeVault RECIPIENT undefined`)
  }

  await deploy({
    hre,
    name: 'SequencerFeeVault',
    args: [sequencerFeeVaultRecipient],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'RECIPIENT',
        ethers.utils.getAddress(sequencerFeeVaultRecipient)
      )
    },
  })
}

deployFn.tags = ['SequencerFeeVaultImpl', 'l2']

export default deployFn
