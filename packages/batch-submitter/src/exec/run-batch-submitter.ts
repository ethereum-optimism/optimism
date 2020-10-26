/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { exit } from 'process'
import { Signer, Wallet } from 'ethers'
import { Provider, JsonRpcProvider } from '@ethersproject/providers'
import { OptimismProvider } from '@eth-optimism/provider'

/* Internal Imports */
import { BatchSubmitter, CanonicalTransactionChainContract } from '..'

/* Logger */
const log = getLogger('oe:batch-submitter:init')

interface RequiredEnvVars {
  SEQUENCER_PRIVATE_KEY: 'SEQUENCER_PRIVATE_KEY'
  L1_NODE_WEB3_URL: 'L1_NODE_WEB3_URL'
  L2_NODE_WEB3_URL: 'L2_NODE_WEB3_URL'
  MAX_TX_SIZE: 'MAX_TX_SIZE'
  POLL_INTERVAL: 'POLL_INTERVAL'
  DEFAULT_BATCH_SIZE: 'DEFAULT_BATCH_SIZE'
  NUM_CONFIRMATIONS: 'NUM_CONFIRMATIONS'
}
const requiredEnvVars: RequiredEnvVars = {
  SEQUENCER_PRIVATE_KEY: 'SEQUENCER_PRIVATE_KEY',
  L1_NODE_WEB3_URL: 'L1_NODE_WEB3_URL',
  L2_NODE_WEB3_URL: 'L2_NODE_WEB3_URL',
  MAX_TX_SIZE: 'MAX_TX_SIZE',
  POLL_INTERVAL: 'POLL_INTERVAL',
  DEFAULT_BATCH_SIZE: 'DEFAULT_BATCH_SIZE',
  NUM_CONFIRMATIONS: 'NUM_CONFIRMATIONS',
}

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

  const batchSubmitter = new BatchSubmitter(
    sequencerSigner,
    l2Provider,
    parseInt(requiredEnvVars.MAX_TX_SIZE, 10),
    parseInt(requiredEnvVars.DEFAULT_BATCH_SIZE, 10),
    parseInt(requiredEnvVars.NUM_CONFIRMATIONS, 10)
  )

  // Run batch submitter!
  while (true) {
    try {
      await batchSubmitter.submitNextBatch()
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
