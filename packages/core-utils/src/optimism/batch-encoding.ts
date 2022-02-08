import { BigNumber, ethers } from 'ethers'

import { add0x, remove0x, encodeHex } from '../common'

export interface BatchContext {
  numSequencedTransactions: number
  numSubsequentQueueTransactions: number
  timestamp: number
  blockNumber: number
}

export interface AppendSequencerBatchParams {
  shouldStartAtElement: number // 5 bytes -- starts at batch
  totalElementsToAppend: number // 3 bytes -- total_elements_to_append
  contexts: BatchContext[] // total_elements[fixed_size[]]
  transactions: string[] // total_size_bytes[],total_size_bytes[]
}

const APPEND_SEQUENCER_BATCH_METHOD_ID = 'appendSequencerBatch()'

export const encodeAppendSequencerBatch = (
  b: AppendSequencerBatchParams
): string => {
  const encodeShouldStartAtElement = encodeHex(b.shouldStartAtElement, 10)
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
    encodeShouldStartAtElement +
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

export const decodeAppendSequencerBatch = (
  b: string
): AppendSequencerBatchParams => {
  b = remove0x(b)

  const shouldStartAtElement = b.slice(0, 10)
  const totalElementsToAppend = b.slice(10, 16)
  const contextHeader = b.slice(16, 22)
  const contextCount = parseInt(contextHeader, 16)

  let offset = 22
  const contexts = []
  for (let i = 0; i < contextCount; i++) {
    const numSequencedTransactions = b.slice(offset, offset + 6)
    offset += 6
    const numSubsequentQueueTransactions = b.slice(offset, offset + 6)
    offset += 6
    const timestamp = b.slice(offset, offset + 10)
    offset += 10
    const blockNumber = b.slice(offset, offset + 10)
    offset += 10
    contexts.push({
      numSequencedTransactions: parseInt(numSequencedTransactions, 16),
      numSubsequentQueueTransactions: parseInt(
        numSubsequentQueueTransactions,
        16
      ),
      timestamp: parseInt(timestamp, 16),
      blockNumber: parseInt(blockNumber, 16),
    })
  }

  const transactions = []
  for (const context of contexts) {
    for (let i = 0; i < context.numSequencedTransactions; i++) {
      const size = b.slice(offset, offset + 6)
      offset += 6
      const raw = b.slice(offset, offset + parseInt(size, 16) * 2)
      transactions.push(add0x(raw))
      offset += raw.length
    }
  }

  return {
    shouldStartAtElement: parseInt(shouldStartAtElement, 16),
    totalElementsToAppend: parseInt(totalElementsToAppend, 16),
    contexts,
    transactions,
  }
}

export const sequencerBatch = {
  encode: (b: AppendSequencerBatchParams) => {
    return (
      ethers.utils.id(APPEND_SEQUENCER_BATCH_METHOD_ID).slice(0, 10) +
      encodeAppendSequencerBatch(b)
    )
  },
  decode: (b: string): AppendSequencerBatchParams => {
    b = remove0x(b)
    const functionSelector = b.slice(0, 8)
    if (
      functionSelector !==
      ethers.utils.id(APPEND_SEQUENCER_BATCH_METHOD_ID).slice(2, 10)
    ) {
      throw new Error('Incorrect function signature')
    }
    return decodeAppendSequencerBatch(b.slice(8))
  },
}
