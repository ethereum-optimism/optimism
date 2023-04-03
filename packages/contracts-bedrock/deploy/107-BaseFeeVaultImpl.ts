import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import { ethers } from 'ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const l1 = hre.network.companionNetworks['l1']
  const deployConfig = hre.getDeployConfig(l1)

  const baseFeeVaultRecipient = deployConfig.baseFeeVaultRecipient
  if (baseFeeVaultRecipient === ethers.constants.AddressZero) {
    throw new Error('BaseFeeVault RECIPIENT undefined')
  }

  await deploy({
    hre,
    name: 'BaseFeeVault',
    args: [baseFeeVaultRecipient],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'RECIPIENT',
        ethers.utils.getAddress(baseFeeVaultRecipient)
      )
    },
  })
}

deployFn.tags = ['BaseFeeVaultImpl', 'l2']

export default deployFn
