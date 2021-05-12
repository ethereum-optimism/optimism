/* Imports: External */
import { injectL2Context } from '@eth-optimism/core-utils'
import hre from 'hardhat'
import { ethers } from 'ethers'

/* Imports: Internal */
import { getContractFactory } from '..'
import { makeActionBundleFromConfig } from './hardhat-tools'

const DEPLOYER_PRIVATE_KEY = process.env.DEPLOYER_PRIVATE_KEY
const L2_NODE_URL = process.env.L2_NODE_URL
const CHUGSPLASH_DEPLOYER_ADDRESS = process.env.CHUGSPLASH_DEPLOYER_ADDRESS
const UPGRADE_CONFIG_PATH = process.env.UPGRADE_CONFIG_PATH

const deploy = async () => {
  const l2Provider = injectL2Context(
    new ethers.providers.JsonRpcProvider(L2_NODE_URL)
  )
  const signer = new ethers.Wallet(DEPLOYER_PRIVATE_KEY, l2Provider)

  const config = require(UPGRADE_CONFIG_PATH)
  const actionBundle = await makeActionBundleFromConfig(hre, config)

  const deployerContract = getContractFactory(
    'L2ChugSplashDeployer',
    signer
  ).attach(CHUGSPLASH_DEPLOYER_ADDRESS)

  while ((await deployerContract.currentBundleHash()) !== actionBundle.root) {
    // Sleep 1 second
    await new Promise((r) => setTimeout(r, 1_000))
  }

  const approveTx = await deployerContract.approveTransactionBundle(
    actionBundle.root,
    actionBundle.actions.length
  )

  await l2Provider.waitForTransaction(approveTx.hash)
}

deploy().catch((err) => {
  console.error(err)
  process.exit(1)
})
