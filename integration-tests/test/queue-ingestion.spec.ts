/* Imports: Internal */
import { injectL2Context } from '@eth-optimism/core-utils'
import { sleep } from './shared/utils'
import { OptimismEnv } from './shared/env'

/* Imports: External */
import { providers } from 'ethers'
import { expect } from 'chai'

// This test ensures that the transactions which get `enqueue`d get
// added to the L2 blocks by the Sync Service (which queries the DTL)
describe('Queue Ingestion', () => {
  const RETRIES = 20
  const numTxs = 5
  let startBlock: number
  let endBlock: number
  let env: OptimismEnv
  let l2Provider: providers.JsonRpcProvider
  const receipts = []

  before(async () => {
    env = await OptimismEnv.new()
    l2Provider = injectL2Context(env.l2Wallet.provider as any)
  })

  // The transactions are enqueue'd with a `to` address of i.repeat(40)
  // meaning that the `to` value is different each iteration in a deterministic
  // way. They need to be inserted into the L2 chain in an ascending order.
  // Keep track of the receipts so that the blockNumber can be compared
  // against the `L1BlockNumber` on the tx objects.
  before(async () => {
    // Keep track of the L2 tip before submitting any transactions so that
    // the subsequent transactions can be queried for in the next test
    startBlock = (await l2Provider.getBlockNumber()) + 1
    endBlock = startBlock + numTxs - 1

    // Enqueue some transactions by building the calldata and then sending
    // the transaction to Layer 1
    for (let i = 0; i < numTxs; i++) {
      const input = ['0x' + `${i}`.repeat(40), 500_000, `0x0${i}`]
      const calldata = env.ctc.interface.encodeFunctionData('enqueue', input)

      const txResponse = await env.l1Wallet.sendTransaction({
        data: calldata,
        to: env.ctc.address,
      })

      const receipt = await txResponse.wait()
      receipts.push(receipt)
    }
  })

  // The batch submitter will notice that there are transactions
  // that are in the queue and submit them. L2 will pick up the
  // sequencer batch appended event and play the transactions.
  it('should order transactions correctly', async () => {
    // Wait until each tx from the previous test has
    // been executed
    let i: number
    for (i = 0; i < RETRIES; i++) {
      const tip = await l2Provider.getBlockNumber()
      if (tip >= endBlock) {
        break
      }
      await sleep(1000)
    }

    if (i === RETRIES) {
      throw new Error(
        'timed out waiting for queued transactions to be inserted'
      )
    }

    const from = await env.l1Wallet.getAddress()
    // Keep track of an index into the receipts list and
    // increment it for each block fetched.
    let receiptIndex = 0
    // Fetch blocks
    for (i = 0; i < numTxs; i++) {
      const block = await l2Provider.getBlock(startBlock + i)
      const hash = block.transactions[0]
      // Use as any hack because additional properties are
      // added to the transaction response
      const tx = await (l2Provider.getTransaction(hash) as any)

      // The `to` addresses are defined in the previous test and
      // increment sequentially.
      expect(tx.to).to.be.equal('0x' + `${i}`.repeat(40))
      // The queue origin is Layer 1
      expect(tx.queueOrigin).to.be.equal('l1')
      // the L1TxOrigin is equal to the Layer one from
      expect(tx.l1TxOrigin).to.be.equal(from.toLowerCase())
      expect(typeof tx.l1BlockNumber).to.be.equal('number')
      // Get the receipt and increment the recept index
      const receipt = receipts[receiptIndex++]
      expect(tx.l1BlockNumber).to.be.equal(receipt.blockNumber)
    }
  })
})
