import { SequencerBatch, BatchType } from '@eth-optimism/core-utils'

import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import { envConfig } from './shared/utils'

describe('Batch Serialization', () => {
  let env: OptimismEnv
  // Allow for each type to be tested. The env var here must be
  // the same value that is passed to the batch submitter
  const batchType = envConfig.BATCH_SUBMITTER_SEQUENCER_BATCH_TYPE.toUpperCase()
  before(async () => {
    env = await OptimismEnv.new()
  })

  it('should fetch batches', async () => {
    const tip = await env.l1Provider.getBlockNumber()
    const ctc = env.messenger.contracts.l1.CanonicalTransactionChain
    const logs = await ctc.queryFilter(
      ctc.filters.TransactionBatchAppended(),
      0,
      tip
    )
    // collect all of the batches
    const batches = []
    for (const log of logs) {
      const tx = await env.l1Provider.getTransaction(log.transactionHash)
      batches.push(tx.data)
    }

    expect(batches.length).to.be.gt(0, 'Submit some batches first')

    let latest = 0
    // decode all of the batches
    for (const batch of batches) {
      // Typings don't work?
      const decoded = (SequencerBatch as any).fromHex(batch)
      expect(decoded.type).to.eq(BatchType[batchType])

      // Iterate over all of the transactions, fetch them
      // by hash and make sure their blocknumbers are in
      // ascending order. This lets us skip handling deposits here
      for (const transaction of decoded.transactions) {
        const tx = transaction.toTransaction()
        const got = await env.l2Provider.getTransaction(tx.hash)
        expect(got).to.not.eq(null)
        expect(got.blockNumber).to.be.gt(latest)
        latest = got.blockNumber
      }
    }
  })
})
