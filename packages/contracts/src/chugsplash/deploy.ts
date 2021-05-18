/* Imports: External */
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { ethers, Signer } from 'ethers'
import { sleep } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { getContractFactory } from '../contract-defs'
import { makeActionBundleFromConfig } from './hardhat-tools'
import { ChugSplashConfig } from './config'

const DEPLOYER_PRIVATE_KEY = process.env.DEPLOYER_PRIVATE_KEY
const L2_NODE_URL = process.env.L2_NODE_URL
const CHUGSPLASH_DEPLOYER_ADDRESS = process.env.CHUGSPLASH_DEPLOYER_ADDRESS
const UPGRADE_CONFIG_PATH = process.env.UPGRADE_CONFIG_PATH

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
  console.log('Loaded config', config)
  const actionBundle = await makeActionBundleFromConfig(args.hre, config)
  console.log('Created action bundle', actionBundle)

  const deployerContract = getContractFactory(
    'L2ChugSplashDeployer',
    args.signer
  ).attach(args.chugsplashDeployerAddress)

  const startTime = Date.now()
  while ((await deployerContract.currentBundleHash()) !== actionBundle.root) {
    const retryIntervalInMs = args.retryIntervalInMs || 1_000 // 1s
    const timeoutInMs = args.timeoutInMs || 600_000 // 10min
    console.log(
      'Action bundle still not active',
      await deployerContract.currentBundleHash(),
      actionBundle.root
    )
    if (Date.now() - startTime > timeoutInMs) {
      console.log('Action bundle not detected, exiting')
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
