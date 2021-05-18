/* Imports: External */
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { ethers, Signer } from 'ethers'
import { sleep } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { getContractFactory } from '../contract-defs'
import { makeActionBundleFromConfig } from './hardhat-tools'
import { ChugSplashConfig } from './config'

export interface executeActionArgs {
  hre: HardhatRuntimeEnvironment
  signer: Signer
  chugsplashDeployerAddress: string
  upgradeConfigPath: string
  timeoutInMs?: number
  retryIntervalInMs?: number
}

export const executeActionsFromConfig = async (
  args: executeActionArgs
): Promise<ethers.providers.TransactionReceipt[]> => {
  const config: ChugSplashConfig = require(args.upgradeConfigPath)
  const actionBundle = await makeActionBundleFromConfig(args.hre, config)

  const deployerContract = getContractFactory(
    'L2ChugSplashDeployer',
    args.signer
  ).attach(args.chugsplashDeployerAddress)

  const startTime = Date.now()
  while ((await deployerContract.currentBundleHash()) !== actionBundle.root) {
    const retryIntervalInMs = args.retryIntervalInMs || 1_000 // 1s
    const timeoutInMs = args.timeoutInMs || 600_000 // 10min
    if (Date.now() - startTime > timeoutInMs) {
      return
    }
    await sleep(retryIntervalInMs)
  }

  const receipts = []
  for (const action of actionBundle.actions) {
    const tx = await deployerContract.executeAction(action.action, action.proof)
    receipts.push(await args.signer.provider.waitForTransaction(tx.hash))
  }

  return receipts
}
