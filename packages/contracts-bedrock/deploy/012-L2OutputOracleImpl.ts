import { DeployFunction } from 'hardhat-deploy/dist/types'
import { constants } from 'ethers'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../scripts/deploy-utils'

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
      await assertContractVariable(contract, 'PROPOSER', constants.AddressZero)
      await assertContractVariable(
        contract,
        'CHALLENGER',
        constants.AddressZero
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
