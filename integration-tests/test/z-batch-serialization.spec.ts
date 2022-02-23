import { sequencerBatch, BatchType } from '@eth-optimism/core-utils'

import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'

describe('Batch Serialization', () => {
  let env: OptimismEnv
  before(async () => {
    env = await OptimismEnv.new()
  })

  it('should fetch batches', async () => {
    const tip = await env.l1Provider.getBlockNumber()
    const logs = await env.ctc.queryFilter(
      env.ctc.filters.TransactionBatchAppended(),
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

    // decode all of the batches
    for (const batch of batches) {
      const decoded = sequencerBatch.decode(batch)
      expect(decoded.type).to.eq(BatchType.ZLIB)
    }
  })
})
