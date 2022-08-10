/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { BigNumber } from 'ethers'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  let deployL2StartingTimestamp = deployConfig.l2OutputOracleStartingTimestamp
  if (deployL2StartingTimestamp < 0) {
    const l1 = hre.ethers.provider
    const l1StartingBlock = await l1.getBlock(deployConfig.l1StartingBlockTag)
    if (l1StartingBlock === null) {
      throw new Error(
        `Cannot fetch block tag ${deployConfig.l1StartingBlockTag}`
      )
    }
    deployL2StartingTimestamp = l1StartingBlock.timestamp
  }

  await deploy('L2OutputOracleProxy', {
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  await deploy('L2OutputOracle', {
    from: deployer,
    args: [
      deployConfig.l2OutputOracleSubmissionInterval,
      deployConfig.l2OutputOracleGenesisL2Output,
      deployConfig.l2OutputOracleHistoricalTotalBlocks,
      deployConfig.l2OutputOracleStartingBlockNumber,
      deployL2StartingTimestamp,
      deployConfig.l2BlockTime,
      deployConfig.l2OutputOracleProposer,
      deployConfig.l2OutputOracleOwner,
    ],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const oracle = await hre.deployments.get('L2OutputOracle')
  const proxy = await hre.deployments.get('L2OutputOracleProxy')
  const Proxy = await hre.ethers.getContractAt('Proxy', proxy.address)

  const L2OutputOracle = await hre.ethers.getContractAt(
    'L2OutputOracle',
    proxy.address
  )

  const tx = await Proxy.upgradeToAndCall(
    oracle.address,
    L2OutputOracle.interface.encodeFunctionData(
      'initialize(bytes32,uint256,address,address)',
      [
        deployConfig.l2OutputOracleGenesisL2Output,
        deployConfig.l2OutputOracleStartingBlockNumber,
        deployConfig.l2OutputOracleProposer,
        deployConfig.l2OutputOracleOwner,
      ]
    )
  )
  await tx.wait()

  const submissionInterval = await L2OutputOracle.SUBMISSION_INTERVAL()
  if (
    !submissionInterval.eq(
      BigNumber.from(deployConfig.l2OutputOracleSubmissionInterval)
    )
  ) {
    throw new Error('submission internal misconfigured')
  }

  const historicalBlocks = await L2OutputOracle.HISTORICAL_TOTAL_BLOCKS()
  if (
    !historicalBlocks.eq(
      BigNumber.from(deployConfig.l2OutputOracleHistoricalTotalBlocks)
    )
  ) {
    throw new Error('historal total blocks misconfigured')
  }

  const startingBlockNumber = await L2OutputOracle.STARTING_BLOCK_NUMBER()
  if (
    !startingBlockNumber.eq(
      BigNumber.from(deployConfig.l2OutputOracleStartingBlockNumber)
    )
  ) {
    throw new Error('starting block number misconfigured')
  }

  const startingTimestamp = await L2OutputOracle.STARTING_TIMESTAMP()
  if (!startingTimestamp.eq(BigNumber.from(deployL2StartingTimestamp))) {
    throw new Error('starting timestamp misconfigured')
  }
  const l2BlockTime = await L2OutputOracle.L2_BLOCK_TIME()
  if (!l2BlockTime.eq(BigNumber.from(deployConfig.l2BlockTime))) {
    throw new Error('L2 block time misconfigured')
  }
}

deployFn.tags = ['L2OutputOracle']

export default deployFn
