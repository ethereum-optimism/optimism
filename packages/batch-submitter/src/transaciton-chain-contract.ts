/* External Imports */
import { Contract, BigNumber } from 'ethers'
import { TransactionResponse } from '@ethersproject/abstract-provider'
import { keccak256 } from 'ethers/lib/utils'
import { remove0x, encodeHex } from './utils'

export interface BatchContext {
  numSequencedTransactions: number
  numSubsequentQueueTransactions: number
  timestamp: number
  blockNumber: number
}

export interface AppendSequencerBatchParams {
  shouldStartAtBatch: number // 5 bytes -- starts at batch
  totalElementsToAppend: number // 3 bytes -- total_elements_to_append
  contexts: BatchContext[] // total_elements[fixed_size[]]
  transactions: string[] // total_size_bytes[],total_size_bytes[]
}

/*
 * OVM_CanonicalTransactionChainContract is a wrapper around a normal Ethers contract
 * where the `appendSequencerBatch(...)` function uses a specialized encoding for improved efficiency.
 */
export class CanonicalTransactionChainContract extends Contract {
  public async appendSequencerBatch(
    batch: AppendSequencerBatchParams
  ): Promise<TransactionResponse> {
    return appendSequencerBatch(this, batch)
  }
}

/**********************
 * Internal Functions *
 *********************/

const APPEND_SEQUENCER_BATCH_METHOD_ID = 'appendSequencerBatch()'

const appendSequencerBatch = async (
  OVM_CanonicalTransactionChain: Contract,
  batch: AppendSequencerBatchParams
): Promise<TransactionResponse> => {
  const methodId = keccak256(
    Buffer.from(APPEND_SEQUENCER_BATCH_METHOD_ID)
  ).slice(2, 10)
  const calldata = encodeAppendSequencerBatch(batch)
  return OVM_CanonicalTransactionChain.signer.sendTransaction({
    to: OVM_CanonicalTransactionChain.address,
    data: '0x' + methodId + calldata,
  })
}

export const encodeAppendSequencerBatch = (
  b: AppendSequencerBatchParams
): string => {
  const encodedShouldStartAtBatch = encodeHex(b.shouldStartAtBatch, 10)
  const encodedTotalElementsToAppend = encodeHex(b.totalElementsToAppend, 6)

  const encodedContextsHeader = encodeHex(b.contexts.length, 6)
  const encodedContexts =
    encodedContextsHeader +
    b.contexts.reduce((acc, cur) => acc + encodeBatchContext(cur), '')

  const encodedTransactionData = b.transactions.reduce((acc, cur) => {
    if (cur.length % 2 !== 0) {
      throw new Error('Unexpected uneven hex string value!')
    }
    const encodedTxDataHeader = remove0x(
      BigNumber.from(remove0x(cur).length / 2).toHexString()
    ).padStart(6, '0')
    return acc + encodedTxDataHeader + remove0x(cur)
  }, '')
  return (
    encodedShouldStartAtBatch +
    encodedTotalElementsToAppend +
    encodedContexts +
    encodedTransactionData
  )
}

const encodeBatchContext = (context: BatchContext): string => {
  return (
    encodeHex(context.numSequencedTransactions, 6) +
    encodeHex(context.numSubsequentQueueTransactions, 6) +
    encodeHex(context.timestamp, 10) +
    encodeHex(context.blockNumber, 10)
  )
}
