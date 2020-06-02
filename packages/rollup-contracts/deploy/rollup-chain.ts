/* External Imports */
import { deploy, deployContract } from '@eth-optimism/core-utils'
import { Wallet } from 'ethers'

/* Internal Imports */
import * as RollupMerkleUtils from '../build/contracts/RollupMerkleUtils.json'
import * as CanonicalTransactionChain from '../build/contracts/CanonicalTransactionChain.json'
import { resolve } from 'path'

const rollupChainDeploymentFunction = async (
  wallet: Wallet
): Promise<string> => {
  console.log(`\nDeploying Rollup Chain!\n`)
  //TODO fix this 
  const l1ToL2TransactionPasser = wallet
  const FORCE_INCLUSION_PERIOD = 600

  const rollupMerkleUtils = await deployContract(RollupMerkleUtils, wallet)

  const canonicalTxChain = await deployContract(
    CanonicalTransactionChain,
    wallet,
    rollupMerkleUtils.address,
    wallet.address,
    l1ToL2TransactionPasser.address,
    FORCE_INCLUSION_PERIOD,
  )

  const l1ToL2QueueAddress = await canonicalTxChain.l1ToL2Queue()
  
  const safetyQueueAddress = await canonicalTxChain.safetyQueue()

  console.log(`Canonical Transaction Chain deployed to ${canonicalTxChain.address}!\n\n`)
  console.log(`Rollup Merkle Utils deployed to ${rollupMerkleUtils.address}!\n\n`)
  console.log(`L1-to-L2 Transaction Queue deployed to ${l1ToL2QueueAddress}!\n\n`)
  console.log(`Safety Transaction Queue deployed to ${safetyQueueAddress}!\n\n`)

  return canonicalTxChain.address
}

/**
 * Deploys the RollupChain contracts.
 *
 * @param rootContract Whether or not this is the main contract being deployed (as compared to a dependency).
 * @returns The deployed contract's address.
 */
export const deployRollupChain = async (
  rootContract: boolean = false
): Promise<string> => {
  // Note: Path is from 'build/deploy/<script>.js'
  const configDirPath = resolve(__dirname, `../../config/`)

  return deploy(rollupChainDeploymentFunction, configDirPath, rootContract)
}

deployRollupChain(true)
