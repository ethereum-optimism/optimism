/* Imports: External */
import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'

task('deploy-teleportr-l1')
  .addParam(
    'minDepositAmountEth',
    'Minimum deposit amount, in ETH.',
    undefined,
    types.string
  )
  .addParam(
    'maxDepositAmountEth',
    'Maximum deposit amount, in ETH.',
    undefined,
    types.string
  )
  .addParam(
    'maxBalanceEth',
    'Maximum contract balance, in ETH.',
    undefined,
    types.string
  )
  .addOptionalParam(
    'numDeployConfirmations',
    'Number of confirmations to wait for each transaction in the deployment. More is safer.',
    1,
    types.int
  )
  .setAction(
    async (
      {
        minDepositAmountEth,
        maxDepositAmountEth,
        maxBalanceEth,
        numDeployConfirmations,
      },
      hre: any
    ) => {
      const { deploy } = hre.deployments
      const { deployer } = await hre.getNamedAccounts()

      console.log('Deploying TeleportrDeposit... ')
      await deploy('TeleportrDeposit', {
        from: deployer,
        args: [
          ethers.utils.parseEther(minDepositAmountEth),
          ethers.utils.parseEther(maxDepositAmountEth),
          ethers.utils.parseEther(maxBalanceEth),
        ],
        log: true,
        waitConfirmations: numDeployConfirmations,
      })
      console.log('Done.')
    }
  )

task('deploy-teleportr-l2').setAction(
  async ({ numDeployConfirmations }, hre: any) => {
    const { deploy } = hre.deployments
    const { deployer } = await hre.getNamedAccounts()

    console.log('Deploying TeleportrDisburser... ')
    await deploy('TeleportrDisburser', {
      from: deployer,
      args: [],
      log: true,
      waitConfirmations: numDeployConfirmations,
    })
    console.log('Done.')
  }
)
