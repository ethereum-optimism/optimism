import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import { ethers } from 'ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const l1 = hre.network.companionNetworks['l1']
  const deployConfig = hre.getDeployConfig(l1)

  const l1FeeVaultRecipient = deployConfig.l1FeeVaultRecipient
  if (l1FeeVaultRecipient === ethers.constants.AddressZero) {
    throw new Error('L1FeeVault RECIPIENT undefined')
  }
  const l1FeeVaultMinimumWithdrawalAmount =
    deployConfig.l1FeeVaultMinimumWithdrawalAmount
  const l1FeeVaultWithdrawalNetwork = deployConfig.l1FeeVaultWithdrawalNetwork
  if (l1FeeVaultWithdrawalNetwork >= 2) {
    throw new Error('L1FeeVault WITHDRAWAL_NETWORK must be 0 or 1')
  }

  await deploy({
    hre,
    name: 'L1FeeVault',
    args: [
      l1FeeVaultRecipient,
      l1FeeVaultMinimumWithdrawalAmount,
      l1FeeVaultWithdrawalNetwork,
    ],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'RECIPIENT',
        ethers.utils.getAddress(l1FeeVaultRecipient)
      )
      await assertContractVariable(
        contract,
        'MIN_WITHDRAWAL_AMOUNT',
        l1FeeVaultMinimumWithdrawalAmount
      )
      await assertContractVariable(
        contract,
        'WITHDRAWAL_NETWORK',
        l1FeeVaultWithdrawalNetwork
      )
    },
  })
}

deployFn.tags = ['L1FeeVaultImpl', 'l2']

export default deployFn
