import zlib from 'zlib'

import { Transaction, parse, serialize } from '@ethersproject/transactions'
import { BigNumber, ethers } from 'ethers'

import { add0x, remove0x, encodeHex } from '../common'

export interface BatchContext {
  numSequencedTransactions: number
  numSubsequentQueueTransactions: number
  timestamp: number
  blockNumber: number
}

export enum BatchType {
  LEGACY = -1,
  ZLIB = 0,
}

export interface AppendSequencerBatchParams {
  shouldStartAtElement: number // 5 bytes -- starts at batch
  totalElementsToAppend: number // 3 bytes -- total_elements_to_append
  contexts: BatchContext[] // total_elements[fixed_size[]]
  transactions: string[] | Transaction[] // total_size_bytes[],total_size_bytes[]
  type?: BatchType
}

export interface EncodeSequencerBatchOptions {
  buffer?: boolean
}

export interface DecodeSequencerBatchOpts {
  decodeTransactions?: boolean
}

const APPEND_SEQUENCER_BATCH_METHOD_ID = 'appendSequencerBatch()'
const FOUR_BYTE_APPEND_SEQUENCER_BATCH = Buffer.from(
  ethers.utils.id(APPEND_SEQUENCER_BATCH_METHOD_ID).slice(2, 10),
  'hex'
)

export const encodeAppendSequencerBatch = (
  b: AppendSequencerBatchParams,
  opts?: EncodeSequencerBatchOptions
): string | Buffer => {
  const encodeShouldStartAtElement = encodeHex(b.shouldStartAtElement, 10)
  const encodedTotalElementsToAppend = encodeHex(b.totalElementsToAppend, 6)
  let contexts = b.contexts.slice()
  let transactions = b.transactions

  if (transactions.length > 0 && typeof transactions[0] !== 'string') {
    const serialized = []
    for (const tx of transactions as Transaction[]) {
      serialized.push(
        serialize(
          {
            nonce: tx.nonce,
            gasPrice: tx.gasPrice,
            gasLimit: tx.gasLimit,
            to: tx.to,
            value: tx.value,
            data: tx.data,
          },
          {
            v: tx.v,
            r: tx.r,
            s: tx.s,
          }
        )
      )
    }
    transactions = serialized
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

  if (b.type === BatchType.ZLIB) {
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

  const ret =
    encodeShouldStartAtElement +
    encodedTotalElementsToAppend +
    encodedContexts +
    encodedTransactionData

  if (opts?.buffer) {
    return Buffer.from(remove0x(ret), 'hex')
  }
  return ret
}

const encodeBatchContext = (context: BatchContext): string => {
  return (
    encodeHex(context.numSequencedTransactions, 6) +
    encodeHex(context.numSubsequentQueueTransactions, 6) +
    encodeHex(context.timestamp, 10) +
    encodeHex(context.blockNumber, 10)
  )
}

const decodeAppendSequencerBatch = (
  b: string | Buffer,
  opts: DecodeSequencerBatchOpts
): AppendSequencerBatchParams => {
  let buf: Buffer
  if (typeof b === 'string') {
    b = remove0x(b)
    buf = Buffer.from(b, 'hex')
  } else {
    buf = b
  }

  const shouldStartAtElement = buf.slice(0, 5)
  const totalElementsToAppend = buf.slice(5, 8)
  const contextHeader = buf.slice(8, 11)
  const contextCount = parseInt(contextHeader.toString('hex'), 8)
  let batchType = BatchType.LEGACY

  let offset = 11
  let contexts = []
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
      numSequencedTransactions: parseInt(
        numSequencedTransactions.toString('hex'),
        16
      ),
      numSubsequentQueueTransactions: parseInt(
        numSubsequentQueueTransactions.toString('hex'),
        16
      ),
      timestamp: parseInt(timestamp.toString('hex'), 16),
      blockNumber: parseInt(blockNumber.toString('hex'), 16),
    })
  }

  if (contexts.length > 0) {
    const context = contexts[0]
    if (context.blockNumber === 0) {
      switch (context.timestamp) {
        case 0: {
          buf = Buffer.concat([
            buf.slice(0, offset),
            zlib.inflateSync(buf.slice(offset)),
          ])
          batchType = BatchType.ZLIB
          break
        }
      }

      // remove the dummy context
      contexts = contexts.slice(1)
    }
  }

  let transactions = []
  for (const context of contexts) {
    for (let i = 0; i < context.numSequencedTransactions; i++) {
      const size = buf.slice(offset, offset + 3)
      offset += 3
      const raw = buf.slice(offset, offset + parseInt(size.toString('hex'), 16))
      transactions.push(add0x(raw.toString('hex')))
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
    shouldStartAtElement: parseInt(shouldStartAtElement.toString('hex'), 16),
    totalElementsToAppend: parseInt(totalElementsToAppend.toString('hex'), 16),
    contexts,
    transactions,
    type: batchType,
  }
}

export const sequencerBatch = {
  encode: (
    b: AppendSequencerBatchParams,
    opts?: EncodeSequencerBatchOptions
  ): string | Buffer => {
    if (opts?.buffer) {
      return Buffer.concat([
        FOUR_BYTE_APPEND_SEQUENCER_BATCH,
        encodeAppendSequencerBatch(b, opts) as Buffer,
      ])
    }
    return ('0x' +
      FOUR_BYTE_APPEND_SEQUENCER_BATCH.toString('hex') +
      encodeAppendSequencerBatch(b, opts)) as string
  },
  decode: (
    b: string | Buffer,
    opts?: DecodeSequencerBatchOpts
  ): AppendSequencerBatchParams => {
    if (typeof b === 'string') {
      b = Buffer.from(remove0x(b), 'hex')
    }
    const fnSelector = b.slice(0, 4)
    if (Buffer.compare(fnSelector, FOUR_BYTE_APPEND_SEQUENCER_BATCH) !== 0) {
      throw new Error(`Incorrect function signature`)
    }
    return decodeAppendSequencerBatch(b.slice(4), opts)
  },
}
