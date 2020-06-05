/* External Imports */
import { deploy, deployContract } from '@eth-optimism/core-utils'
import { Wallet } from 'ethers'
import { Provider } from 'ethers/providers'

/* Internal Imports */
import * as RollupMerkleUtils from '../build/RollupMerkleUtils.json'
import * as CanonicalTransactionChain from '../build/CanonicalTransactionChain.json'
import * as StateCommitmentChain from '../build/StateCommitmentChain.json'
import * as SequencerBatchSubmitter from '../build/SequencerBatchSubmitter.json'

import { resolve } from 'path'

const rollupChainDeploymentFunction = async (
  wallet: Wallet,
  provider: Provider
): Promise<string> => {
  console.log(`\nDeploying Rollup Chain!\n`)

  const l1ToL2TransactionPasser = wallet // TODO actually deploy l1ToL2TransactionPasser
  const sequencer = process.env.SEQUENCER_PRIVATE_KEY
    ? new Wallet(process.env.SEQUENCER_PRIVATE_KEY, provider)
    : wallet
  const inclusionPeriod = process.env.FORCE_INCLUSION_PERIOD || 600
  const fraudVerifier = wallet // TODO actually deploy Fraud Verifier

  console.log(`Deploying RollupMerkleUtils...`)
  const rollupMerkleUtils = await deployContract(RollupMerkleUtils, wallet)

  console.log(`Deploying SequencerBatchSubmitter...`)
  const sequencerBatchSubmitter = await deployContract(
    SequencerBatchSubmitter,
    wallet,
    sequencer.address
  )

  console.log(`Deploying CanonicalTransactionChain...`)
  const canonicalTxChain = await deployContract(
    CanonicalTransactionChain,
    wallet,
    rollupMerkleUtils.address,
    sequencerBatchSubmitter.address,
    l1ToL2TransactionPasser.address,
    inclusionPeriod
  )

  const l1ToL2QueueAddress = await canonicalTxChain.l1ToL2Queue()

  const safetyQueueAddress = await canonicalTxChain.safetyQueue()

  console.log(`Deploying StateCommitmentChain...`)
  const stateChain = await deployContract(
    StateCommitmentChain,
    wallet,
    rollupMerkleUtils.address,
    canonicalTxChain.address,
    fraudVerifier.address
  )

  console.log(`Initializing SequencerBatchSubmitter with chain addresses...`)
  await sequencerBatchSubmitter
    .connect(sequencer)
    .initialize(canonicalTxChain.address, stateChain.address)

  console.log(
    `\nRollup Merkle Utils deployed to ${rollupMerkleUtils.address}!\n`
  )
  console.log(
    `Canonical Transaction Chain deployed to ${canonicalTxChain.address}!\n`
  )
  console.log(`L1-to-L2 Transaction Queue deployed to ${l1ToL2QueueAddress}!\n`)
  console.log(`Safety Transaction Queue deployed to ${safetyQueueAddress}!\n`)
  console.log(`State Commitment Chain deployed to ${stateChain.address}!\n`)
  console.log(
    `Sequencer Batch Submitter deployed to ${sequencerBatchSubmitter.address}!\n`
  )
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
