import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { names } from '../src/address-names'
import { getContractFromArtifact } from '../src/hardhat-deploy-ethers'
import { AppendSequencerBatchParams, remove0x, add0x } from '@eth-optimism/core-utils'

// start block index
// end block index

const decodeAppendSequencerBatch = (
  b: string
): AppendSequencerBatchParams => {
  b = remove0x(b)
  const buf = Buffer.from(b, 'hex')

  const shouldStartAtElement = buf.slice(0, 5)
  const totalElementsToAppend = buf.slice(5, 8)
  const contextHeader = buf.slice(8, 11)
  const contextCount = parseInt(contextHeader.toString('hex'), 8)

  let offset = 11
  const contexts = []
  for (let i = 0; i < contextCount; i++) {
    const numSequencedTransactions = buf.slice(offset, offset + 3)
    offset += 3
    const numSubsequentQueueTransactions = buf.slice(offset, offset + 3)
    offset += 3
    const timestamp = buf.slice(offset, offset + 5)
    offset += 5
    const blockNumber = buf.slice(offset, offset + 5)
    offset += 5
    contexts.push({
      numSequencedTransactions: parseInt(numSequencedTransactions.toString('hex'), 16),
      numSubsequentQueueTransactions: parseInt(
        numSubsequentQueueTransactions.toString('hex'),
        16
      ),
      timestamp: parseInt(timestamp.toString('hex'), 16),
      blockNumber: parseInt(blockNumber.toString('hex'), 16),
    })
  }

  const transactions = []
  for (const context of contexts) {
    for (let i = 0; i < context.numSequencedTransactions; i++) {
      const size = buf.slice(offset, offset + 3)
      offset += 3
      const raw = buf.slice(offset, offset + parseInt(size.toString('hex'), 16))
      transactions.push(add0x(raw.toString('hex')))
      offset += raw.length
    }
  }

  return {
    shouldStartAtElement: parseInt(shouldStartAtElement.toString('hex'), 16),
    totalElementsToAppend: parseInt(totalElementsToAppend.toString('hex'), 16),
    contexts,
    transactions,
  }
}

task('fetch-batches')
  .addOptionalParam(
    'contractsRpcUrl',
    'Sequencer HTTP Endpoint',
    process.env.CONTRACTS_RPC_URL,
    types.string
  )
  .setAction(async (args, hre) => {
    // get all sequencer batch appended events
    // get all transactions
    const provider = new ethers.providers.JsonRpcProvider(args.contractsRpcUrl)

    const CanonicalTransactionChain = await getContractFromArtifact(
      hre,
      names.managed.contracts.CanonicalTransactionChain, {
        signerOrProvider: provider,
      }
    )

    const start = 13596466
    const end = start + 1200

    const events = await CanonicalTransactionChain.queryFilter(
      CanonicalTransactionChain.filters.SequencerBatchAppended(),
      start,
      end
    )

    const batches = []
    for (const event of events) {
      const tx = await provider.getTransaction(event.transactionHash)
      console.log('batch deserialize')
      const batch = decodeAppendSequencerBatch(tx.data)
      batches.push(batch)
    }

    // deserialize the txs
    for (const batch of batches) {
      console.log(batch)
    }
  })
