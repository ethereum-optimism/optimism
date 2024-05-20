import { DeployFunction } from 'hardhat-deploy/dist/types'
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
    args: [],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'SUBMISSION_INTERVAL', 1)
      await assertContractVariable(contract, 'L2_BLOCK_TIME', 1)
      await assertContractVariable(contract, 'startingBlockNumber', 0)
      await assertContractVariable(contract, 'startingTimestamp', 0)
    },
  })
}

deployFn.tags = ['L2OutputOracleImpl', 'setup', 'l1']

export default deployFn
