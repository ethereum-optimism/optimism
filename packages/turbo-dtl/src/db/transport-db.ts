import {
  BatchEntry,
  BatchTransactionEntry,
  EnqueueLinkEntry,
  EnqueueTransactionEntry,
} from './entries'
import { IndexLike, SimpleDB } from './simple-db'
import { BSS_HF1_BLOCK, PATCH_CONTEXTS } from './patches'

export enum Keys {
  ENQUEUE_TRANSACTION,
  BATCHED_TRANSACTION,
  BATCH,
  HIGHEST_SYNCED_L1_BLOCK,
  HIGHEST_SYNCED_L2_BLOCK,
  HIGHEST_KNOWN_L2_BLOCK,
  ENQUEUE_LINK,
}

export class TransportDB {
  constructor(public db: SimpleDB, public l2ChainId: number) {}

  public async getHighestSyncedL2Block(): Promise<number | null> {
    return this.db.get(Keys.HIGHEST_SYNCED_L2_BLOCK, 'latest')
  }

  public async putHighestSyncedL2Block(val: number): Promise<void> {
    return this.db.put(Keys.HIGHEST_SYNCED_L2_BLOCK, 'latest', val)
  }

  public async getHighestKnownL2Block(): Promise<number | null> {
    return this.db.get(Keys.HIGHEST_KNOWN_L2_BLOCK, 'latest')
  }

  public async putHighestKnownL2Block(val: number): Promise<void> {
    return this.db.put(Keys.HIGHEST_KNOWN_L2_BLOCK, 'latest', val)
  }

  public async getHighestSyncedL1Block(): Promise<number | null> {
    return this.db.get(Keys.HIGHEST_SYNCED_L1_BLOCK, 'latest')
  }

  public async putHighestSyncedL1Block(val: number): Promise<void> {
    return this.db.put(Keys.HIGHEST_SYNCED_L1_BLOCK, 'latest', val)
  }

  public async getEnqueue(
    index: IndexLike
  ): Promise<EnqueueTransactionEntry | null> {
    const enqueue = await this.db.get(Keys.ENQUEUE_TRANSACTION, index)
    if (enqueue === null) {
      return null
    }

    const link = await this.getEnqueueLink(enqueue.index)
    if (link === null) {
      return null
    }

    return {
      ...enqueue,
      ctcIndex: link.chainIndex,
    }
  }

  public async putEnqueue(val: EnqueueTransactionEntry): Promise<void> {
    return this.db.put(Keys.ENQUEUE_TRANSACTION, val.index, val)
  }

  public async getEnqueueLink(
    index: IndexLike
  ): Promise<EnqueueLinkEntry | null> {
    return this.db.get(Keys.ENQUEUE_LINK, index)
  }

  public async putEnqueueLink(
    val: EnqueueLinkEntry,
    index: IndexLike
  ): Promise<void> {
    return this.db.put(Keys.ENQUEUE_LINK, index, val)
  }

  public async getTransaction(
    index: IndexLike
  ): Promise<BatchTransactionEntry | null> {
    const transaction = await this.db.get(Keys.BATCHED_TRANSACTION, index)
    if (transaction === null) {
      return null
    }

    if (transaction.queueOrigin !== 'l1') {
      return transaction
    }

    const enqueue = await this.getEnqueue(transaction.queueIndex)
    if (enqueue === null) {
      return null
    }

    let timestamp = enqueue.timestamp
    if (transaction.index >= (BSS_HF1_BLOCK[this.l2ChainId] || 0)) {
      timestamp = transaction.timestamp
    }

    const patches = PATCH_CONTEXTS[this.l2ChainId]
    if (patches && patches[transaction.index + 1]) {
      timestamp = patches[transaction.index + 1]
    }

    return {
      ...transaction,
      ...{
        blockNumber: enqueue.blockNumber,
        timestamp,
        gasLimit: enqueue.gasLimit,
        target: enqueue.target,
        origin: enqueue.origin,
        data: enqueue.data,
      },
    }
  }

  public async putTransaction(val: BatchTransactionEntry): Promise<void> {
    return this.db.put(Keys.BATCHED_TRANSACTION, val.index, val)
  }

  public async getBatch(index: IndexLike): Promise<BatchEntry | null> {
    return this.db.get(Keys.BATCH, index)
  }

  public async putBatch(val: BatchEntry): Promise<void> {
    return this.db.put(Keys.BATCH, val.index, val)
  }
}
