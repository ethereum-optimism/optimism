import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  let BatchInboxProxy: ethers.Contract
  try {
    BatchInboxProxy = await getContractFromArtifact(
      hre,
      'Proxy__OVM_BatchInbox'
    )
  } catch {
    BatchInboxProxy = await getContractFromArtifact(
      hre,
      'BatchInboxProxy'
    )
  }

  await deployAndVerifyAndThen({
    hre,
    name: 'BatchInbox',
    args: [BatchInboxProxy.address],
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

deployFn.tags = ['BatchInboxImpl', 'fresh']

export default deployFn
