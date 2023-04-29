import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  if (hre.deployConfig.l2BlockTime === 0) {
    throw new Error(
      'L2OutputOracle deployment: l2BlockTime must be greater than 0'
    )
  } else if (
    hre.deployConfig.l2OutputOracleSubmissionInterval <=
    hre.deployConfig.l2BlockTime
  ) {
    throw new Error(
      'L2OutputOracle deployment: submissionInterval must be greater than the l2BlockTime'
    )
  }

  await deploy({
    hre,
    name: 'L2OutputOracle',
    args: [
      hre.deployConfig.l2OutputOracleSubmissionInterval,
      hre.deployConfig.l2BlockTime,
      0,
      0,
      hre.deployConfig.l2OutputOracleProposer,
      hre.deployConfig.l2OutputOracleChallenger,
      hre.deployConfig.finalizationPeriodSeconds,
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
        'PROPOSER',
        hre.deployConfig.l2OutputOracleProposer
      )
      await assertContractVariable(
        contract,
        'CHALLENGER',
        hre.deployConfig.l2OutputOracleChallenger
      )
      await assertContractVariable(
        contract,
        'FINALIZATION_PERIOD_SECONDS',
        hre.deployConfig.finalizationPeriodSeconds
      )
    },
  })
}

deployFn.tags = ['L2OutputOracleImpl', 'setup', 'l1']

export default deployFn
