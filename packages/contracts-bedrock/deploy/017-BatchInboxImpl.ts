import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import {
  assertContractVariable,
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  let BatchInboxProxy: ethers.Contract
  BatchInboxProxy = await getContractFromArtifact(
    hre,
    'BatchInboxProxy'
  )

  await deployAndVerifyAndThen({
    hre,
    name: 'BatchInbox',
    args: [hre.deployConfig.batchSenderAddress, hre.deployConfig.batchInboxAddress],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'proposer',
        hre.deployConfig.batchSenderAddress
      )
      await assertContractVariable(
        contract,
        'owner',
        hre.deployConfig.batchInboxAddress
      )
    },
  })
}

deployFn.tags = ['BatchInboxImpl']

export default deployFn
