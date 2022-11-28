import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import {
  assertContractVariable,
  deployAndVerifyAndThen,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deployAndVerifyAndThen({
    hre,
    name: 'L2OutputOracle',
    args: [
      hre.deployConfig.l2OutputOracleSubmissionInterval,
      hre.deployConfig.l2BlockTime,
      0,
      0,
      hre.deployConfig.l2OutputOracleProposer,
      hre.deployConfig.l2OutputOracleOwner,
    ],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'SUBMISSION_INTERVAL',
        hre.deployConfig.l2OutputOracleSubmissionInterval
      )
      await assertContractVariable(
        contract,
        'L2_BLOCK_TIME',
        hre.deployConfig.l2BlockTime
      )
      await assertContractVariable(
        contract,
        'proposer',
        hre.deployConfig.l2OutputOracleProposer
      )
      await assertContractVariable(
        contract,
        'owner',
        hre.deployConfig.l2OutputOracleOwner
      )
    },
  })
}

deployFn.tags = ['L2OutputOracleImpl']

export default deployFn
