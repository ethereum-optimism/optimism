import { Transaction, parse } from '@ethersproject/transactions'
import { BigNumber, ethers } from 'ethers'

import zlib from 'zlib'

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
  transactions: string[] | Transaction[] // total_size_bytes[],total_size_bytes[]
}

export interface EncodeSequencerBatchOptions {
  zlib?: boolean
}

export interface DecodeSequencerBatchOpts {
  decodeTransactions?: boolean
}

const APPEND_SEQUENCER_BATCH_METHOD_ID = 'appendSequencerBatch()'

export const encodeAppendSequencerBatch = (
  b: AppendSequencerBatchParams,
  opts?: EncodeSequencerBatchOptions
): string => {
  const encodeShouldStartAtElement = encodeHex(b.shouldStartAtElement, 10)
  const encodedTotalElementsToAppend = encodeHex(b.totalElementsToAppend, 6)
  let contexts = b.contexts.slice()
  const transactions = b.transactions

  if (transactions.length > 0 && typeof transactions[0] !== 'string') {
    // TODO: flatten the transactions into strings
    throw new Error('Must pass in serialized transactions')
  }

  let encodedTransactionData = (transactions as string[]).reduce((acc, cur) => {
    if (cur.length % 2 !== 0) {
      throw new Error('Unexpected uneven hex string value!')
    }
    const encodedTxDataHeader = remove0x(
      BigNumber.from(remove0x(cur).length / 2).toHexString()
    ).padStart(6, '0')
    return acc + encodedTxDataHeader + remove0x(cur)
  }, '')

  if (opts?.zlib) {
    const compressed = zlib
      .deflateSync(Buffer.from(encodedTransactionData, 'hex'))
      .toString('hex')

    encodedTransactionData = compressed
    contexts = [
      {
        numSequencedTransactions: 0,
        numSubsequentQueueTransactions: 0,
        timestamp: 0,
        blockNumber: 0,
      },
      ...contexts,
    ]
  }

  const encodedContextsHeader = encodeHex(contexts.length, 6)
  const encodedContexts =
    encodedContextsHeader +
    contexts.reduce((acc, cur) => acc + encodeBatchContext(cur), '')

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
  b: string,
  opts?: DecodeSequencerBatchOpts
): AppendSequencerBatchParams => {
  b = remove0x(b)

  const shouldStartAtElement = b.slice(0, 10)
  const totalElementsToAppend = b.slice(10, 16)
  const contextHeader = b.slice(16, 22)
  const contextCount = parseInt(contextHeader, 16)

  let offset = 22
  let contexts = []
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

  if (contexts.length > 0) {
    const context = contexts[0]
    if (context.blockNumber === 0) {
      switch (context.timestamp) {
        case 0: {
          b =
            b.slice(0, offset) +
            zlib
              .inflateSync(Buffer.from(b.slice(offset), 'hex'))
              .toString('hex')
          break
        }
      }

      // remove the dummy context
      contexts = contexts.slice(1)
    }
  }

  let transactions = []
  for (const context of contexts) {
    for (let j = 0; j < context.numSequencedTransactions; j++) {
      const size = b.slice(offset, offset + 6)
      offset += 6
      const raw = b.slice(offset, offset + parseInt(size, 16) * 2)
      transactions.push(add0x(raw))
      offset += raw.length
    }
  }

  if (opts?.decodeTransactions) {
    const decoded = []
    for (const tx of transactions) {
      const parsed = parse(tx)
      decoded.push(parsed)
    }
    transactions = decoded
  }

  return {
    shouldStartAtElement: parseInt(shouldStartAtElement, 16),
    totalElementsToAppend: parseInt(totalElementsToAppend, 16),
    contexts,
    transactions,
  }
}

export const sequencerBatch = {
  encode: (
    b: AppendSequencerBatchParams,
    opts?: EncodeSequencerBatchOptions
  ) => {
    return (
      ethers.utils.id(APPEND_SEQUENCER_BATCH_METHOD_ID).slice(0, 10) +
      encodeAppendSequencerBatch(b, opts)
    )
  },
  decode: (
    b: string,
    opts?: DecodeSequencerBatchOpts
  ): AppendSequencerBatchParams => {
    b = remove0x(b)
    const functionSelector = b.slice(0, 8)
    if (
      functionSelector !==
      ethers.utils.id(APPEND_SEQUENCER_BATCH_METHOD_ID).slice(2, 10)
    ) {
      throw new Error('Incorrect function signature')
    }
    return decodeAppendSequencerBatch(b.slice(8), opts)
  },
}
