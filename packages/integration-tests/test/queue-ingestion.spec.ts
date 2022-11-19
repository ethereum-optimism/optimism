/* Imports: Internal */
import { providers } from 'ethers'
import { applyL1ToL2Alias } from '@eth-optimism/core-utils'
import { asL2Provider } from '@eth-optimism/sdk'
import { getContractInterface } from '@eth-optimism/contracts'

/* Imports: External */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import { DEFAULT_TEST_GAS_L1, envConfig } from './shared/utils'

describe('Queue Ingestion', () => {
  let env: OptimismEnv
  let l2Provider: providers.JsonRpcProvider
  before(async () => {
    env = await OptimismEnv.new()
    l2Provider = asL2Provider(env.l2Wallet.provider as any)
  })

  // The batch submitter will notice that there are transactions
  // that are in the queue and submit them. L2 will pick up the
  // sequencer batch appended event and play the transactions.
  it('should order transactions correctly', async () => {
    const numTxs = envConfig.OVMCONTEXT_SPEC_NUM_TXS

    // Enqueue some transactions by building the calldata and then sending
    // the transaction to Layer 1
    const txs = []
    for (let i = 0; i < numTxs; i++) {
      const tx =
        await env.messenger.contracts.l1.L1CrossDomainMessenger.sendMessage(
          `0x${`${i}`.repeat(40)}`,
          `0x0${i}`,
          1_000_000,
          {
            gasLimit: DEFAULT_TEST_GAS_L1,
          }
        )
      await tx.wait()
      txs.push(tx)
    }

    for (let i = 0; i < numTxs; i++) {
      const l1Tx = txs[i]
      const l1TxReceipt = await txs[i].wait()
      const receipt = await env.waitForXDomainTransaction(l1Tx)
      const l2Tx = (await l2Provider.getTransaction(
        receipt.remoteTx.hash
      )) as any

      const params = getContractInterface(
        'L2CrossDomainMessenger'
      ).decodeFunctionData('relayMessage', l2Tx.data)

      expect(params._sender.toLowerCase()).to.equal(
        env.l1Wallet.address.toLowerCase()
      )
      expect(params._target).to.equal('0x' + `${i}`.repeat(40))
      expect(l2Tx.queueOrigin).to.equal('l1')
      expect(l2Tx.l1TxOrigin.toLowerCase()).to.equal(
        applyL1ToL2Alias(
          env.messenger.contracts.l1.L1CrossDomainMessenger.address
        ).toLowerCase()
      )
      expect(l2Tx.l1BlockNumber).to.equal(l1TxReceipt.blockNumber)
    }
  })
})
