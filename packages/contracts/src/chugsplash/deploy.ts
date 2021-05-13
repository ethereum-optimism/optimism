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

export const executeActionsFromConfig = async (
  hre: HardhatRuntimeEnvironment,
  signer: Signer,
  chugsplashDeployerAddress: string,
  upgradeConfigPath: string
): Promise<ethers.providers.TransactionReceipt[]> => {
  const config: ChugSplashConfig = require(upgradeConfigPath)
  console.log('Loaded config', config)
  const actionBundle = await makeActionBundleFromConfig(hre, config)
  console.log('Created action bundle', actionBundle)

  const deployerContract = getContractFactory(
    'L2ChugSplashDeployer',
    signer
  ).attach(chugsplashDeployerAddress)

  while ((await deployerContract.currentBundleHash()) !== actionBundle.root) {
    console.log(
      'Action bundle still not active',
      await deployerContract.currentBundleHash(),
      actionBundle.root
    )
    await sleep(1_000)
  }

  const receipts = []
  for (const action of actionBundle.actions) {
    const tx = await deployerContract.executeAction(action.action, action.proof)
    receipts.push(await signer.provider.waitForTransaction(tx.hash))
  }

  return receipts
}

const deploy = async () => {
  const l2Provider = new ethers.providers.JsonRpcProvider(L2_NODE_URL)
  const signer = new ethers.Wallet(DEPLOYER_PRIVATE_KEY, l2Provider)

  const receipts = []
  // const receipts = await executeActionsFromConfig(
  //   signer,
  //   CHUGSPLASH_DEPLOYER_ADDRESS,
  //   UPGRADE_CONFIG_PATH
  // )

  console.log('Executed actions', receipts)
}
