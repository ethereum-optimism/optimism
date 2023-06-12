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
  const sequencerFeeVaultMinimumWithdrawalAmount =
    deployConfig.sequencerFeeVaultMinimumWithdrawalAmount
  const sequencerFeeVaultWithdrawalNetwork =
    deployConfig.sequencerFeeVaultWithdrawalNetwork
  if (sequencerFeeVaultWithdrawalNetwork >= 2) {
    throw new Error('SequencerFeeVault WITHDRAWAL_NETWORK must be 0 or 1')
  }

  await deploy({
    hre,
    name: 'SequencerFeeVault',
    args: [
      sequencerFeeVaultRecipient,
      sequencerFeeVaultMinimumWithdrawalAmount,
      sequencerFeeVaultWithdrawalNetwork,
    ],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'RECIPIENT',
        ethers.utils.getAddress(sequencerFeeVaultRecipient)
      )
      await assertContractVariable(
        contract,
        'MIN_WITHDRAWAL_AMOUNT',
        sequencerFeeVaultMinimumWithdrawalAmount
      )
      await assertContractVariable(
        contract,
        'WITHDRAWAL_NETWORK',
        sequencerFeeVaultWithdrawalNetwork
      )
    },
  })
}

deployFn.tags = ['SequencerFeeVaultImpl', 'l2']

export default deployFn
