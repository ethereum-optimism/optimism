/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  if (
    typeof hre.deployConfig.startingTimestamp !== 'number' ||
    isNaN(hre.deployConfig.startingTimestamp)
  ) {
    throw new Error(
      'Cannot deploy L2OutputOracle without specifying a valid startingTimestamp.'
    )
  }

  await deploy('L2OutputOracle', {
    from: deployer,
    args: [
      hre.deployConfig.submissionInterval,
      hre.deployConfig.genesisOutput,
      hre.deployConfig.historicalBlocks,
      hre.deployConfig.startingBlockNumber,
      hre.deployConfig.startingTimestamp,
      hre.deployConfig.l2BlockTime,
      hre.deployConfig.sequencerAddress,
    ],
    log: true,
    waitConfirmations: 1,
  })
}

deployFn.tags = ['L2OutputOracle']

export default deployFn
