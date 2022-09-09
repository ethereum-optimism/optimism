import zlib from 'zlib'

import { parse, serialize, Transaction } from '@ethersproject/transactions'
import { Struct, BufferWriter, BufferReader } from 'bufio'
import { id } from '@ethersproject/hash'

import { remove0x } from '../common'

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
  transactions: string[] // total_size_bytes[],total_size_bytes[]
  type?: BatchType
}

const APPEND_SEQUENCER_BATCH_METHOD_ID = 'appendSequencerBatch()'
const FOUR_BYTE_APPEND_SEQUENCER_BATCH = Buffer.from(
  id(APPEND_SEQUENCER_BATCH_METHOD_ID).slice(2, 10),
  'hex'
)

// Legacy support
// This function returns the serialized batch
// without the 4 byte selector and without the
// 0x prefix
export const encodeAppendSequencerBatch = (
  b: AppendSequencerBatchParams
): string => {
  for (const tx of b.transactions) {
    if (tx.length % 2 !== 0) {
      throw new Error('Unexpected uneven hex string value!')
    }
  }
  const batch = sequencerBatch.encode(b)
  const fnSelector = batch.slice(2, 10)
  if (fnSelector !== FOUR_BYTE_APPEND_SEQUENCER_BATCH.toString('hex')) {
    throw new Error(`Incorrect function signature`)
  }
  return batch.slice(10)
}

// Legacy support
// This function assumes there is no 4byte selector
// as part of the input data
export const decodeAppendSequencerBatch = (
  b: string
): AppendSequencerBatchParams => {
  const calldata =
    '0x' + FOUR_BYTE_APPEND_SEQUENCER_BATCH.toString('hex') + remove0x(b)
  return sequencerBatch.decode(calldata)
}

// Legacy support
export const sequencerBatch = {
  encode: (params: AppendSequencerBatchParams): string => {
    const batch = new SequencerBatch({
      shouldStartAtElement: params.shouldStartAtElement,
      totalElementsToAppend: params.totalElementsToAppend,
      contexts: params.contexts.map((c) => new Context(c)),
      transactions: params.transactions.map((t) =>
        BatchedTx.fromTransaction(t)
      ),
      type: params.type,
    })
    return batch.toHex()
  },
  decode: (b: string): AppendSequencerBatchParams => {
    const buf = Buffer.from(remove0x(b), 'hex')
    const fnSelector = buf.slice(0, 4)
    if (Buffer.compare(fnSelector, FOUR_BYTE_APPEND_SEQUENCER_BATCH) !== 0) {
      throw new Error(`Incorrect function signature`)
    }

    const batch = SequencerBatch.decode<SequencerBatch>(buf)
    const params: AppendSequencerBatchParams = {
      shouldStartAtElement: batch.shouldStartAtElement,
      totalElementsToAppend: batch.totalElementsToAppend,
      contexts: batch.contexts.map((c) => ({
        numSequencedTransactions: c.numSequencedTransactions,
        numSubsequentQueueTransactions: c.numSubsequentQueueTransactions,
        timestamp: c.timestamp,
        blockNumber: c.blockNumber,
      })),
      transactions: batch.transactions.map((t) => t.toHexTransaction()),
      type: batch.type,
    }

    return params
  },
}

export class Context extends Struct {
  // 3 bytes
  public numSequencedTransactions: number = 0
  // 3 bytes
  public numSubsequentQueueTransactions: number = 0
  // 5 bytes
  public timestamp: number = 0
  // 5 bytes
  public blockNumber: number = 0

  constructor(options: Partial<Context> = {}) {
    super()

    if (typeof options.numSequencedTransactions === 'number') {
      this.numSequencedTransactions = options.numSequencedTransactions
    }
    if (typeof options.numSubsequentQueueTransactions === 'number') {
      this.numSubsequentQueueTransactions =
        options.numSubsequentQueueTransactions
    }
    if (typeof options.timestamp === 'number') {
      this.timestamp = options.timestamp
    }
    if (typeof options.blockNumber === 'number') {
      this.blockNumber = options.blockNumber
    }
  }

  getSize(): number {
    return 16
  }

  write(bw: BufferWriter): BufferWriter {
    bw.writeU24BE(this.numSequencedTransactions)
    bw.writeU24BE(this.numSubsequentQueueTransactions)
    bw.writeU40BE(this.timestamp)
    bw.writeU40BE(this.blockNumber)
    return bw
  }

  read(br: BufferReader): this {
    this.numSequencedTransactions = br.readU24BE()
    this.numSubsequentQueueTransactions = br.readU24BE()
    this.timestamp = br.readU40BE()
    this.blockNumber = br.readU40BE()
    return this
  }

  toJSON() {
    return {
      numSequencedTransactions: this.numSequencedTransactions,
      numSubsequentQueueTransactions: this.numSubsequentQueueTransactions,
      timestamp: this.timestamp,
      blockNumber: this.blockNumber,
    }
  }
}

// transaction
export class BatchedTx extends Struct {
  // 3 bytes
  public txSize: number
  // rlp encoded transaction
  public raw: Buffer
  public tx: Transaction

  constructor(tx?: Transaction) {
    super()
    this.tx = tx
  }

  getSize(): number {
    if (this.raw && this.raw.length) {
      return this.raw.length + 3
    }
    const tx = serialize(
      {
        nonce: this.tx.nonce,
        gasPrice: this.tx.gasPrice,
        gasLimit: this.tx.gasLimit,
        to: this.tx.to,
        value: this.tx.value,
        data: this.tx.data,
      },
      {
        v: this.tx.v,
        r: this.tx.r,
        s: this.tx.s,
      }
    )

    // remove 0x prefix
    this.raw = Buffer.from(remove0x(tx), 'hex')
    return this.raw.length + 3
  }

  write(bw: BufferWriter): BufferWriter {
    bw.writeU24BE(this.txSize)
    bw.writeBytes(this.raw)
    return bw
  }

  read(br: BufferReader): this {
    this.txSize = br.readU24BE()
    this.raw = br.readBytes(this.txSize)
    return this
  }

  toTransaction(): Transaction {
    if (this.tx) {
      return this.tx
    }
    return parse(this.raw)
  }

  toHexTransaction(): string {
    if (this.raw) {
      return '0x' + this.raw.toString('hex')
    }
    return serialize(
      {
        nonce: this.tx.nonce,
        gasPrice: this.tx.gasPrice,
        gasLimit: this.tx.gasLimit,
        to: this.tx.to,
        value: this.tx.value,
        data: this.tx.data,
      },
      {
        v: this.tx.v,
        r: this.tx.r,
        s: this.tx.s,
      }
    )
  }

  toJSON() {
    if (!this.tx) {
      this.tx = parse(this.raw)
    }

    return {
      nonce: this.tx.nonce,
      gasPrice: this.tx.gasPrice.toString(),
      gasLimit: this.tx.gasLimit.toString(),
      to: this.tx.to,
      value: this.tx.value.toString(),
      data: this.tx.data,
      v: this.tx.v,
      r: this.tx.r,
      s: this.tx.s,
      chainId: this.tx.chainId,
      hash: this.tx.hash,
      from: this.tx.from,
    }
  }

  // TODO: inconsistent API with toTransaction
  // but unnecessary right now
  // this should be fromHexTransaction
  fromTransaction(tx: string): this {
    this.raw = Buffer.from(remove0x(tx), 'hex')
    this.txSize = this.raw.length
    return this
  }

  fromHex(s: string, extra?: object): this {
    const buffer = Buffer.from(remove0x(s), 'hex')
    return this.decode(buffer, extra)
  }

  static fromTransaction(s: string) {
    return new this().fromTransaction(s)
  }
}

export class SequencerBatch extends Struct {
  // 5 bytes
  public shouldStartAtElement: number
  // 3 bytes
  public totalElementsToAppend: number
  // 3 byte header for count, []Context
  public contexts: Context[]
  // []3 byte size, rlp encoded tx
  public transactions: BatchedTx[]

  // The batch type that determines how
  // it is serialized
  public type: BatchType

  constructor(options: Partial<SequencerBatch> = {}) {
    super()
    this.contexts = []
    this.transactions = []

    if (typeof options.shouldStartAtElement === 'number') {
      this.shouldStartAtElement = options.shouldStartAtElement
    }
    if (typeof options.totalElementsToAppend === 'number') {
      this.totalElementsToAppend = options.totalElementsToAppend
    }
    if (Array.isArray(options.contexts)) {
      this.contexts = options.contexts
    }
    if (Array.isArray(options.transactions)) {
      this.transactions = options.transactions
    }
    if (typeof options.type === 'number') {
      this.type = options.type
    }
  }

  write(bw: BufferWriter): BufferWriter {
    bw.writeBytes(FOUR_BYTE_APPEND_SEQUENCER_BATCH)

    bw.writeU40BE(this.shouldStartAtElement)
    bw.writeU24BE(this.totalElementsToAppend)

    const contexts = this.contexts.slice()
    if (this.type === BatchType.ZLIB) {
      contexts.unshift(
        new Context({
          blockNumber: 0,
          timestamp: 0,
          numSequencedTransactions: 0,
          numSubsequentQueueTransactions: 0,
        })
      )
    }
    bw.writeU24BE(contexts.length)

    for (const context of contexts) {
      context.write(bw)
    }

    if (this.type === BatchType.ZLIB) {
      const writer = new BufferWriter()
      for (const tx of this.transactions) {
        tx.write(writer)
      }
      const compressed = zlib.deflateSync(writer.render())
      bw.writeBytes(compressed)
    } else {
      // Legacy
      for (const tx of this.transactions) {
        tx.write(bw)
      }
    }

    return bw
  }

  read(br: BufferReader): this {
    const selector = br.readBytes(4)
    if (Buffer.compare(selector, FOUR_BYTE_APPEND_SEQUENCER_BATCH) !== 0) {
      br.seek(-4)
    }

    this.type = BatchType.LEGACY
    this.shouldStartAtElement = br.readU40BE()
    this.totalElementsToAppend = br.readU24BE()

    const contexts = br.readU24BE()
    for (let i = 0; i < contexts; i++) {
      const context = Context.read<Context>(br)
      this.contexts.push(context)
    }

    // handle typed batches
    if (this.contexts.length > 0 && this.contexts[0].timestamp === 0) {
      switch (this.contexts[0].blockNumber) {
        case 0: {
          this.type = BatchType.ZLIB
          const bytes = br.readBytes(br.left())
          const inflated = zlib.inflateSync(bytes)
          br = new BufferReader(inflated)

          // remove the dummy context
          this.contexts = this.contexts.slice(1)
          break
        }
      }
    }

    for (const context of this.contexts) {
      for (let i = 0; i < context.numSequencedTransactions; i++) {
        const tx = BatchedTx.read<BatchedTx>(br)
        this.transactions.push(tx)
      }
    }

    return this
  }

  getSize(): number {
    if (this.type === BatchType.ZLIB) {
      return -1
    }

    let size = 8 + 3 + 4
    for (const context of this.contexts) {
      size += context.getSize()
    }

    for (const tx of this.transactions) {
      size += tx.getSize()
    }
    return size
  }

  fromHex(s: string, extra?: object): this {
    const buffer = Buffer.from(remove0x(s), 'hex')
    return this.decode(buffer, extra)
  }

  toHex(): string {
    return '0x' + this.encode().toString('hex')
  }

  toJSON() {
    return {
      shouldStartAtElement: this.shouldStartAtElement,
      totalElementsToAppend: this.totalElementsToAppend,
      contexts: this.contexts.map((c) => c.toJSON()),
      transactions: this.transactions.map((tx) => tx.toJSON()),
    }
  }
}
