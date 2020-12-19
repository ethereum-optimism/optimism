/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { exit } from 'process'
import { Signer, Wallet } from 'ethers'
import {
  Provider,
  JsonRpcProvider,
  TransactionReceipt,
} from '@ethersproject/providers'
import { OptimismProvider } from '@eth-optimism/provider'
import { config } from 'dotenv'
config()

/* Internal Imports */
import {
  TransactionBatchSubmitter,
  StateBatchSubmitter,
  STATE_BATCH_SUBMITTER_LOG_TAG,
  TX_BATCH_SUBMITTER_LOG_TAG,
} from '..'

/* Logger */
const log = getLogger('oe:batch-submitter:init')

interface RequiredEnvVars {
  SEQUENCER_PRIVATE_KEY: 'SEQUENCER_PRIVATE_KEY'
  L1_NODE_WEB3_URL: 'L1_NODE_WEB3_URL'
  L2_NODE_WEB3_URL: 'L2_NODE_WEB3_URL'
  MIN_TX_SIZE: 'MIN_TX_SIZE'
  MAX_TX_SIZE: 'MAX_TX_SIZE'
  MAX_BATCH_SIZE: 'MAX_BATCH_SIZE'
  POLL_INTERVAL: 'POLL_INTERVAL'
  NUM_CONFIRMATIONS: 'NUM_CONFIRMATIONS'
  FINALITY_CONFIRMATIONS: 'FINALITY_CONFIRMATIONS'
  RUN_TX_BATCH_SUBMITTER: 'true' | 'false' | 'RUN_TX_BATCH_SUBMITTER'
  RUN_STATE_BATCH_SUBMITTER: 'true' | 'false' | 'RUN_STATE_BATCH_SUBMITTER'
}
const requiredEnvVars: RequiredEnvVars = {
  SEQUENCER_PRIVATE_KEY: 'SEQUENCER_PRIVATE_KEY',
  L1_NODE_WEB3_URL: 'L1_NODE_WEB3_URL',
  L2_NODE_WEB3_URL: 'L2_NODE_WEB3_URL',
  MIN_TX_SIZE: 'MIN_TX_SIZE',
  MAX_TX_SIZE: 'MAX_TX_SIZE',
  MAX_BATCH_SIZE: 'MAX_BATCH_SIZE',
  POLL_INTERVAL: 'POLL_INTERVAL',
  NUM_CONFIRMATIONS: 'NUM_CONFIRMATIONS',
  FINALITY_CONFIRMATIONS: 'FINALITY_CONFIRMATIONS',
  RUN_TX_BATCH_SUBMITTER: 'RUN_TX_BATCH_SUBMITTER',
  RUN_STATE_BATCH_SUBMITTER: 'RUN_STATE_BATCH_SUBMITTER',
}
/* Optional Env Vars
 * FRAUD_SUBMISSION_ADDRESS
 * DISABLE_QUEUE_BATCH_APPEND
 */

export const run = async () => {
  log.info('Starting batch submitter...')

  for (const [i, val] of Object.entries(requiredEnvVars)) {
    if (!process.env[val]) {
      log.error(`Missing enviornment variable: ${val}`)
      exit(1)
    }
    requiredEnvVars[val] = process.env[val]
  }

  const l1Provider: Provider = new JsonRpcProvider(
    requiredEnvVars.L1_NODE_WEB3_URL
  )
  const l2Provider: OptimismProvider = new OptimismProvider(
    requiredEnvVars.L2_NODE_WEB3_URL
  )
  const sequencerSigner: Signer = new Wallet(
    requiredEnvVars.SEQUENCER_PRIVATE_KEY,
    l1Provider
  )

  const txBatchSubmitter = new TransactionBatchSubmitter(
    sequencerSigner,
    l2Provider,
    parseInt(requiredEnvVars.MIN_TX_SIZE, 10),
    parseInt(requiredEnvVars.MAX_TX_SIZE, 10),
    parseInt(requiredEnvVars.MAX_BATCH_SIZE, 10),
    parseInt(requiredEnvVars.NUM_CONFIRMATIONS, 10),
    parseInt(requiredEnvVars.NUM_CONFIRMATIONS, 10),
    true,
    getLogger(TX_BATCH_SUBMITTER_LOG_TAG),
    !!process.env.DISABLE_QUEUE_BATCH_APPEND
  )

  const stateBatchSubmitter = new StateBatchSubmitter(
    sequencerSigner,
    l2Provider,
    parseInt(requiredEnvVars.MIN_TX_SIZE, 10),
    parseInt(requiredEnvVars.MAX_TX_SIZE, 10),
    parseInt(requiredEnvVars.MAX_BATCH_SIZE, 10),
    parseInt(requiredEnvVars.NUM_CONFIRMATIONS, 10),
    parseInt(requiredEnvVars.FINALITY_CONFIRMATIONS, 10),
    true,
    getLogger(STATE_BATCH_SUBMITTER_LOG_TAG),
    process.env.FRAUD_SUBMISSION_ADDRESS || 'no fraud'
  )

  // Loops infinitely!
  const loop = async (
    func: () => Promise<TransactionReceipt>
  ): Promise<void> => {
    while (true) {
      try {
        await func()
      } catch (err) {
        log.error('Error submitting batch', err)
        log.info('Retrying...')
      }
      // Sleep
      await new Promise((r) =>
        setTimeout(r, parseInt(requiredEnvVars.POLL_INTERVAL, 10))
      )
    }
  }

  // Run batch submitters in two seperate infinite loops!
  if (requiredEnvVars.RUN_TX_BATCH_SUBMITTER === 'true') {
    loop(() => txBatchSubmitter.submitNextBatch())
  }
  if (requiredEnvVars.RUN_STATE_BATCH_SUBMITTER === 'true') {
    loop(() => stateBatchSubmitter.submitNextBatch())
  }
}
