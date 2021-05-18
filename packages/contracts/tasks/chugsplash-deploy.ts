/* Imports: External */
import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'

import { executeActionsFromConfig } from '../src'

task('chugsplash-deploy', 'Deploys an action bundle to L2')
  .addParam('l2NodeUrl', 'Url of L2 node', types.string)
  // To be converted to ledger
  .addParam('deployerPrivateKey', 'Private key to call execute action', types.string)
  .addParam('chugsplashDeployerAddress', 'Address of the ChugSplash deployer contract', types.string)
  .addParam('upgradeConfigPath', 'Path to upgrade configuration JSON file', types.inputFile)
  .addOptionalParam('timeoutInMs', 'Amount of time to wait before timing out of deploy in milliseconds', types.int)
  .addOptionalParam('retryIntervalInMs', 'Amount of time to wait before checking for new action bundle', types.int)
  .setAction(async (args, hre: any) => {
    const l2Provider = new ethers.providers.JsonRpcProvider(args.l2NodeUrl)
    const signer = new ethers.Wallet(args.deployerPrivateKey, l2Provider)

    return await executeActionsFromConfig({
      hre,
      signer,
      chugsplashDeployerAddress: args.chugsplashDeployerAddress,
      upgradeConfigPath: args.upgradeConfigPath,
      timeoutInMs: args.timeoutInMs,
      retryIntervalInMs: args.retryIntervalInMs
    })
})