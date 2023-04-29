import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  await deploy({
    hre,
    name: 'DisputeGameFactory',
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

deployFn.tags = ['DisputeGameFactoryImpl', 'setup', 'l1']

export default deployFn
