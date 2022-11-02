import assert from 'assert'

import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import {
  assertContractVariable,
  deployAndVerifyAndThen,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  // Use default starting time if starting time is not provided.
  let deployL2StartingTimestamp =
    hre.deployConfig.l2OutputOracleStartingTimestamp
  if (deployL2StartingTimestamp < 0) {
    const l1StartingBlock = await hre.ethers.provider.getBlock(
      hre.deployConfig.l1StartingBlockTag
    )
    if (l1StartingBlock === null) {
      throw new Error(
        `Cannot fetch block tag ${hre.deployConfig.l1StartingBlockTag}`
      )
    }
    deployL2StartingTimestamp = l1StartingBlock.timestamp
  }

  await deployAndVerifyAndThen({
    hre,
    name: 'L2OutputOracle',
    args: [
      hre.deployConfig.l2OutputOracleSubmissionInterval,
      hre.deployConfig.l2OutputOracleGenesisL2Output,
      hre.deployConfig.l2OutputOracleHistoricalTotalBlocks,
      hre.deployConfig.l2OutputOracleStartingBlockNumber,
      deployL2StartingTimestamp,
      hre.deployConfig.l2BlockTime,
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
        'STARTING_BLOCK_NUMBER',
        hre.deployConfig.l2OutputOracleStartingBlockNumber
      )
      await assertContractVariable(
        contract,
        'HISTORICAL_TOTAL_BLOCKS',
        hre.deployConfig.l2OutputOracleHistoricalTotalBlocks
      )
      await assertContractVariable(
        contract,
        'STARTING_TIMESTAMP',
        deployL2StartingTimestamp
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

      // Has to be done separately since l2Output is a mapping.
      if (
        hre.deployConfig.l2OutputOracleGenesisL2Output ===
        ethers.constants.HashZero
      ) {
        console.log(
          `[WARNING] Genesis L2 output is ZERO and should NOT BE ZERO if you are deploying to prod`
        )
      } else {
        const output = await contract.getL2Output(
          hre.deployConfig.l2OutputOracleStartingBlockNumber
        )
        assert(
          output.outputRoot === hre.deployConfig.l2OutputOracleGenesisL2Output,
          `[FATAL] L2OutputOracleImpl genesis output is ${output.outputRoot} but should be ${hre.deployConfig.l2OutputOracleGenesisL2Output}`
        )
      }
    },
  })
}

deployFn.tags = ['L2OutputOracleImpl', 'fresh', 'migration']

export default deployFn
