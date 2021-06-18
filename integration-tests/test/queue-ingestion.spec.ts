import { expect } from 'chai'

/* Imports: Internal */
import { providers } from 'ethers'
import { injectL2Context } from '@eth-optimism/core-utils'

/* Imports: External */
import { OptimismEnv } from './shared/env'
import { Direction } from './shared/watcher-utils'

describe('Queue Ingestion', () => {
  let env: OptimismEnv
  let l2Provider: providers.JsonRpcProvider
  before(async () => {
    env = await OptimismEnv.new()
    l2Provider = injectL2Context(env.l2Wallet.provider as any)
  })

  // The batch submitter will notice that there are transactions
  // that are in the queue and submit them. L2 will pick up the
  // sequencer batch appended event and play the transactions.
  it('should order transactions correctly', async () => {
    const numTxs = 5

    // Enqueue some transactions by building the calldata and then sending
    // the transaction to Layer 1
    const txs = []
    for (let i = 0; i < numTxs; i++) {
      const tx = await env.l1Messenger.sendMessage(
        `0x${`${i}`.repeat(40)}`,
        `0x0${i}`,
        1_000_000
      )
      await tx.wait()
      txs.push(tx)
    }

    for (let i = 0; i < numTxs; i++) {
      const l1Tx = txs[i]
      const l1TxReceipt = await txs[i].wait()
      const receipt = await env.waitForXDomainTransaction(
        l1Tx,
        Direction.L1ToL2
      )
      const l2Tx = (await l2Provider.getTransaction(
        receipt.remoteTx.hash
      )) as any

      const params = env.l2Messenger.interface.decodeFunctionData(
        'relayMessage',
        l2Tx.data
      )

      expect(params._sender.toLowerCase()).to.equal(
        env.l1Wallet.address.toLowerCase()
      )
      expect(params._target).to.equal('0x' + `${i}`.repeat(40))
      expect(l2Tx.queueOrigin).to.equal('l1')
      expect(l2Tx.l1TxOrigin.toLowerCase()).to.equal(
        env.l1Messenger.address.toLowerCase()
      )
      expect(l2Tx.l1BlockNumber).to.equal(l1TxReceipt.blockNumber)
    }
  })
})
