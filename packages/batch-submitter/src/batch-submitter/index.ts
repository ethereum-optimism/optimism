export * from './batch-submitter'
export * from './tx-batch-submitter'
export * from './state-batch-submitter'

export const TX_BATCH_SUBMITTER_LOG_TAG = 'oe:batch-submitter:tx-chain'
export const STATE_BATCH_SUBMITTER_LOG_TAG = 'oe:batch-submitter:state-chain'

// BLOCK_OFFSET is the number of L2 blocks we need to skip for the
// batch submitter.
export const BLOCK_OFFSET = 1 // TODO: Update testnet / mainnet to make this zero.
